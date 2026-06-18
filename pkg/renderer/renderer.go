package renderer

import (
	"fmt"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/cherevatovm/graphics-csm-project/pkg/camera"
	"github.com/cherevatovm/graphics-csm-project/pkg/scene"
	"github.com/cherevatovm/graphics-csm-project/pkg/shader"
	"github.com/cherevatovm/graphics-csm-project/pkg/shadow"
)

type MainRenderer struct {
	FBOs       *shadow.FBOManager
	Calculator *shadow.CascadeCalculator
	Program    shader.Program

	uModelLoc               int32
	uViewLoc                int32
	uProjectionLoc          int32
	uSunDirLoc              int32
	uViewPosLoc             int32
	uCascadeFarPlanesLoc    int32
	uCascadeCountLoc        int32
	uDebugModeLoc           int32
	uObjectColorLoc         int32
	uShadowMapResolutionLoc int32
	uPCFSamplesLoc          int32
	uPCFKernelRadiiLoc      int32
}

func NewMainRenderer(
	fbo *shadow.FBOManager,
	calc *shadow.CascadeCalculator,
	program shader.Program,
) *MainRenderer {
	return &MainRenderer{
		FBOs:                    fbo,
		Calculator:              calc,
		Program:                 program,
		uModelLoc:               program.UniformLocation("uModel"),
		uViewLoc:                program.UniformLocation("uView"),
		uProjectionLoc:          program.UniformLocation("uProjection"),
		uSunDirLoc:              program.UniformLocation("uSunDir"),
		uViewPosLoc:             program.UniformLocation("uViewPos"),
		uCascadeFarPlanesLoc:    program.UniformLocation("uCascadeFarPlanes"),
		uCascadeCountLoc:        program.UniformLocation("uCascadeCount"),
		uDebugModeLoc:           program.UniformLocation("uDebugMode"),
		uObjectColorLoc:         program.UniformLocation("uObjectColor"),
		uShadowMapResolutionLoc: program.UniformLocation("uShadowMapResolution"),
		uPCFSamplesLoc:          program.UniformLocation("uPCFSamples"),
		uPCFKernelRadiiLoc:      program.UniformLocation("uPCFKernelRadii"),
	}
}

func (mr *MainRenderer) Render(sc *scene.Scene, cam *camera.Camera, sunDir mgl32.Vec3, debugMode bool) {
	mr.Program.Use()
	view := cam.ViewMatrix()
	proj := cam.ProjectionMatrix()

	gl.UniformMatrix4fv(mr.uViewLoc, 1, false, &view[0])
	gl.UniformMatrix4fv(mr.uProjectionLoc, 1, false, &proj[0])

	sunDirNorm := sunDir.Normalize()
	gl.Uniform3f(mr.uSunDirLoc, sunDirNorm.X(), sunDirNorm.Y(), sunDirNorm.Z())
	gl.Uniform3f(mr.uViewPosLoc, cam.Position.X(), cam.Position.Y(), cam.Position.Z())

	res := float32(mr.FBOs.Resolution())
	gl.Uniform1f(mr.uShadowMapResolutionLoc, res)

	gl.Uniform1i(mr.uPCFSamplesLoc, int32(shadow.VOGELSampleCount))
	kernelRadii := [4]float32{
		shadow.VOGELKernelRadii0,
		shadow.VOGELKernelRadii1,
		shadow.VOGELKernelRadii2,
		shadow.VOGELKernelRadii3,
	}
	gl.Uniform1fv(mr.uPCFKernelRadiiLoc, 4, &kernelRadii[0])

	farPlanes := make([]float32, mr.Calculator.Config.Count)
	for i := 0; i < mr.Calculator.Config.Count; i++ {
		farPlanes[i] = mr.Calculator.Cascades[i].FarPlane
	}
	gl.Uniform1fv(mr.uCascadeFarPlanesLoc, int32(len(farPlanes)), &farPlanes[0])
	gl.Uniform1i(mr.uCascadeCountLoc, int32(mr.Calculator.Config.Count))

	var debugInt int32
	if debugMode {
		debugInt = 1
	}
	gl.Uniform1i(mr.uDebugModeLoc, debugInt)

	for i := 0; i < mr.Calculator.Config.Count; i++ {
		name := fmt.Sprintf("uLightViewProj[%d]\x00", i)
		loc := gl.GetUniformLocation(uint32(mr.Program), gl.Str(name))
		lvp := mr.Calculator.Cascades[i].LightViewProj
		gl.UniformMatrix4fv(loc, 1, false, &lvp[0])
	}

	shadowTexStartUnit := 2
	mr.FBOs.BindDepthTextures(shadowTexStartUnit)

	for i := 0; i < mr.Calculator.Config.Count; i++ {
		name := fmt.Sprintf("uShadowMaps[%d]\x00", i)
		loc := gl.GetUniformLocation(uint32(mr.Program), gl.Str(name))
		gl.Uniform1i(loc, int32(shadowTexStartUnit+i))
	}

	for _, m := range sc.Meshes {
		gl.UniformMatrix4fv(mr.uModelLoc, 1, false, &m.ModelMatrix[0])
		gl.Uniform3f(mr.uObjectColorLoc, m.Color.X(), m.Color.Y(), m.Color.Z())
		m.Draw()
	}

	if sc.Terrain != nil {
		gl.UniformMatrix4fv(mr.uModelLoc, 1, false, &sc.Terrain.Mesh.ModelMatrix[0])
		gl.Uniform3f(mr.uObjectColorLoc, 1.0, 1.0, 1.0)
		sc.Terrain.Draw()
	}
}
