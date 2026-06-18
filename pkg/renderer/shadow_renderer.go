package renderer

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/cherevatovm/graphics-csm-project/pkg/camera"
	"github.com/cherevatovm/graphics-csm-project/pkg/scene"
	"github.com/cherevatovm/graphics-csm-project/pkg/shader"
	"github.com/cherevatovm/graphics-csm-project/pkg/shadow"
)

type ShadowRenderer struct {
	FBOs       *shadow.FBOManager
	Calculator *shadow.CascadeCalculator
	Program    shader.Program

	uModelLoc         int32
	uLightViewProjLoc int32
}

func NewShadowRenderer(
	fboManager *shadow.FBOManager,
	cascadeCalc *shadow.CascadeCalculator,
	program shader.Program,
) *ShadowRenderer {
	return &ShadowRenderer{
		FBOs:              fboManager,
		Calculator:        cascadeCalc,
		Program:           program,
		uModelLoc:         program.UniformLocation("uModel"),
		uLightViewProjLoc: program.UniformLocation("uLightViewProj"),
	}
}

func (sr *ShadowRenderer) RenderPass(sc *scene.Scene, cam *camera.Camera, sunDir mgl32.Vec3) {
	sr.Calculator.Calculate(cam, sunDir)
	sr.Program.Use()

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.FRONT)

	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.PolygonOffset(shadow.DepthBiasFactor, shadow.DepthBiasUnits)

	for i := 0; i < sr.Calculator.Config.Count; i++ {
		sr.FBOs.BindForWriting(i)
		gl.Clear(gl.DEPTH_BUFFER_BIT)

		lvp := sr.Calculator.Cascades[i].LightViewProj
		gl.UniformMatrix4fv(sr.uLightViewProjLoc, 1, false, &lvp[0])
		for _, m := range sc.Meshes {
			gl.UniformMatrix4fv(sr.uModelLoc, 1, false, &m.ModelMatrix[0])
			m.Draw()
		}

		if sc.Terrain != nil {
			gl.UniformMatrix4fv(sr.uModelLoc, 1, false, &sc.Terrain.Mesh.ModelMatrix[0])
			sc.Terrain.Draw()
		}
	}

	gl.CullFace(gl.BACK)
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}
