package renderer

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/cherevatovm/graphics-csm-project/pkg/camera"
	"github.com/cherevatovm/graphics-csm-project/pkg/shader"
	"github.com/cherevatovm/graphics-csm-project/pkg/shadow"
)

type DebugOverlay struct {
	Enabled     bool
	LineProgram shader.Program
	lineVAO     uint32
	lineVBO     uint32
}

func NewDebugOverlay(lineProgram shader.Program) *DebugOverlay {
	return &DebugOverlay{
		Enabled:     false,
		LineProgram: lineProgram,
	}
}

func (d *DebugOverlay) Toggle() bool {
	d.Enabled = !d.Enabled
	return d.Enabled
}

func (d *DebugOverlay) DrawFrustums(
	calc *shadow.CascadeCalculator,
	cam *camera.Camera,
	sunDir mgl32.Vec3,
) {
	if !d.Enabled {
		return
	}

	colors := shadow.GetCascadeColors()

	for i := 0; i < calc.Config.Count; i++ {
		near := calc.Config.CameraNear
		far := calc.Cascades[i].FarPlane
		if i > 0 {
			near = calc.Cascades[i-1].FarPlane
		}
		corners := calc.GetFrustumCorners(near, far, cam)

		lines := frustumEdges(corners)
		d.drawLines(lines, colors[i], cam)
	}
}

func frustumEdges(corners [8]mgl32.Vec3) []float32 {

	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}

	lines := make([]float32, 0, len(edges)*6)
	for _, e := range edges {
		a, b := corners[e[0]], corners[e[1]]
		lines = append(lines, a.X(), a.Y(), a.Z(), b.X(), b.Y(), b.Z())
	}
	return lines
}

func (d *DebugOverlay) drawLines(lines []float32, color mgl32.Vec3, cam *camera.Camera) {
	if len(lines) == 0 {
		return
	}

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(lines)*4, gl.Ptr(lines), gl.STATIC_DRAW)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 3*4, 0)
	gl.EnableVertexAttribArray(0)

	d.LineProgram.Use()

	view := cam.ViewMatrix()
	proj := cam.ProjectionMatrix()
	d.LineProgram.SetMat4("uView", &view)
	d.LineProgram.SetMat4("uProjection", &proj)

	locColor := d.LineProgram.UniformLocation("uColor")
	gl.Uniform4f(locColor, color.X(), color.Y(), color.Z(), 0.8)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.LineWidth(2.0)

	gl.DrawArrays(gl.LINES, 0, int32(len(lines)/3))

	gl.LineWidth(1.0)
	gl.BindVertexArray(0)

	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteBuffers(1, &vbo)
}
