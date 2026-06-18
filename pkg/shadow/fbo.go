package shadow

import (
	"fmt"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type ShadowFBO struct {
	FBO        uint32
	DepthMap   uint32
	Resolution int32
}

type FBOManager struct {
	FBOs         []ShadowFBO
	CascadeCount int
}

func NewFBOManager(cascadeCount int, resolution int32) (*FBOManager, error) {
	manager := &FBOManager{
		FBOs:         make([]ShadowFBO, cascadeCount),
		CascadeCount: cascadeCount,
	}

	for i := 0; i < cascadeCount; i++ {
		fbo, err := createShadowFBO(resolution)
		if err != nil {
			manager.Release()
			return nil, fmt.Errorf("создание FBO для каскада %d: %w", i, err)
		}
		manager.FBOs[i] = fbo
	}

	return manager, nil
}

func createShadowFBO(resolution int32) (ShadowFBO, error) {
	var sfbo ShadowFBO
	sfbo.Resolution = resolution

	gl.GenTextures(1, &sfbo.DepthMap)
	gl.BindTexture(gl.TEXTURE_2D, sfbo.DepthMap)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.DEPTH_COMPONENT24,
		resolution,
		resolution,
		0,
		gl.DEPTH_COMPONENT,
		gl.FLOAT,
		nil,
	)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

	borderColor := [4]float32{1.0, 1.0, 1.0, 1.0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_MODE, gl.COMPARE_REF_TO_TEXTURE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_FUNC, gl.LEQUAL)

	gl.GenFramebuffers(1, &sfbo.FBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, sfbo.FBO)

	gl.FramebufferTexture2D(
		gl.FRAMEBUFFER,
		gl.DEPTH_ATTACHMENT,
		gl.TEXTURE_2D,
		sfbo.DepthMap,
		0,
	)

	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)

	status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if status != gl.FRAMEBUFFER_COMPLETE {
		gl.DeleteFramebuffers(1, &sfbo.FBO)
		gl.DeleteTextures(1, &sfbo.DepthMap)
		return ShadowFBO{}, fmt.Errorf("FBO не полный, статус: 0x%x", status)
	}

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	return sfbo, nil
}

func (m *FBOManager) BindForWriting(cascadeIndex int) {
	if cascadeIndex < 0 || cascadeIndex >= m.CascadeCount {
		return
	}
	fbo := m.FBOs[cascadeIndex]
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo.FBO)
	gl.Viewport(0, 0, fbo.Resolution, fbo.Resolution)
}

func (m *FBOManager) BindDepthTextures(startUnit int) int {
	for i := 0; i < m.CascadeCount; i++ {
		unit := uint32(gl.TEXTURE0 + startUnit + i)
		gl.ActiveTexture(unit)
		gl.BindTexture(gl.TEXTURE_2D, m.FBOs[i].DepthMap)
	}
	return m.CascadeCount
}

func (m *FBOManager) Resolution() int32 {
	if m.CascadeCount > 0 {
		return m.FBOs[0].Resolution
	}
	return 0
}

func (m *FBOManager) Release() {
	for i := 0; i < m.CascadeCount; i++ {
		if m.FBOs[i].FBO != 0 {
			gl.DeleteFramebuffers(1, &m.FBOs[i].FBO)
		}
		if m.FBOs[i].DepthMap != 0 {
			gl.DeleteTextures(1, &m.FBOs[i].DepthMap)
		}
	}
	m.FBOs = nil
	m.CascadeCount = 0
}

func CheckGLError() error {
	err := gl.GetError()
	if err != gl.NO_ERROR {
		return fmt.Errorf("ошибка OpenGL: 0x%x", err)
	}
	return nil
}
