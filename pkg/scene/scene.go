package scene

import (
	"math"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Mesh struct {
	VAO         uint32
	VBO         uint32
	EBO         uint32
	IndexCount  int32
	ModelMatrix mgl32.Mat4
	Color       mgl32.Vec3
}

type Scene struct {
	Meshes  []*Mesh
	Terrain *Terrain
}

type Vertex struct {
	PosX, PosY, PosZ    float32
	NormX, NormY, NormZ float32
}

const VertexStride = 6 * 4

func NewScene() *Scene {
	return &Scene{
		Meshes: make([]*Mesh, 0),
	}
}

func (s *Scene) AddMesh(m *Mesh) {
	s.Meshes = append(s.Meshes, m)
}

func (s *Scene) Draw() {
	for _, m := range s.Meshes {
		m.Draw()
	}
	if s.Terrain != nil {
		s.Terrain.Draw()
	}
}

func (m *Mesh) Draw() {
	gl.BindVertexArray(m.VAO)
	gl.DrawElements(gl.TRIANGLES, m.IndexCount, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
}

func (m *Mesh) DrawWithModel(setModelUniform func(mat *mgl32.Mat4)) {
	setModelUniform(&m.ModelMatrix)
	m.Draw()
}

func (m *Mesh) Release() {
	if m.VAO != 0 {
		gl.DeleteVertexArrays(1, &m.VAO)
	}
	if m.VBO != 0 {
		gl.DeleteBuffers(1, &m.VBO)
	}
	if m.EBO != 0 {
		gl.DeleteBuffers(1, &m.EBO)
	}
}

func (s *Scene) ReleaseAll() {
	for _, m := range s.Meshes {
		m.Release()
	}
	if s.Terrain != nil {
		s.Terrain.Release()
	}
}

func NewCube() *Mesh {
	vertices := []float32{
		-0.5, -0.5, 0.5, 0, 0, 1,
		0.5, -0.5, 0.5, 0, 0, 1,
		0.5, 0.5, 0.5, 0, 0, 1,
		-0.5, 0.5, 0.5, 0, 0, 1,

		0.5, -0.5, -0.5, 0, 0, -1,
		-0.5, -0.5, -0.5, 0, 0, -1,
		-0.5, 0.5, -0.5, 0, 0, -1,
		0.5, 0.5, -0.5, 0, 0, -1,

		-0.5, 0.5, 0.5, 0, 1, 0,
		0.5, 0.5, 0.5, 0, 1, 0,
		0.5, 0.5, -0.5, 0, 1, 0,
		-0.5, 0.5, -0.5, 0, 1, 0,

		-0.5, -0.5, -0.5, 0, -1, 0,
		0.5, -0.5, -0.5, 0, -1, 0,
		0.5, -0.5, 0.5, 0, -1, 0,
		-0.5, -0.5, 0.5, 0, -1, 0,

		0.5, -0.5, 0.5, 1, 0, 0,
		0.5, -0.5, -0.5, 1, 0, 0,
		0.5, 0.5, -0.5, 1, 0, 0,
		0.5, 0.5, 0.5, 1, 0, 0,

		-0.5, -0.5, -0.5, -1, 0, 0,
		-0.5, -0.5, 0.5, -1, 0, 0,
		-0.5, 0.5, 0.5, -1, 0, 0,
		-0.5, 0.5, -0.5, -1, 0, 0,
	}

	indices := []uint32{
		0, 1, 2, 2, 3, 0,
		4, 5, 6, 6, 7, 4,
		8, 9, 10, 10, 11, 8,
		12, 13, 14, 14, 15, 12,
		16, 17, 18, 18, 19, 16,
		20, 21, 22, 22, 23, 20,
	}

	return createMesh(vertices, indices)
}

func NewCone(radius, height float32, segments int) *Mesh {
	halfH := height / 2.0
	angleStep := 2.0 * math.Pi / float64(segments)

	var apexNormalSum mgl32.Vec3
	sideNormals := make([]mgl32.Vec3, segments)

	rimPos := make([]mgl32.Vec3, segments)
	for i := 0; i < segments; i++ {
		angle := float32(i) * float32(angleStep)
		x := radius * float32(math.Cos(float64(angle)))
		z := radius * float32(math.Sin(float64(angle)))
		rimPos[i] = mgl32.Vec3{x, -halfH, z}
		tangent := mgl32.Vec3{-z, 0, x}
		generatrix := mgl32.Vec3{-x, height, -z}
		sideNormals[i] = generatrix.Cross(tangent).Normalize()
		apexNormalSum = apexNormalSum.Add(sideNormals[i])
	}

	apexNormal := apexNormalSum.Mul(1.0 / float32(segments)).Normalize()
	vertices := make([]float32, 0, (2+2*segments)*6)
	vertices = append(vertices, 0, halfH, 0, apexNormal.X(), apexNormal.Y(), apexNormal.Z())
	for i := 0; i < segments; i++ {
		n := sideNormals[i]
		vertices = append(vertices, rimPos[i].X(), rimPos[i].Y(), rimPos[i].Z(), n.X(), n.Y(), n.Z())
	}

	vertices = append(vertices, 0, -halfH, 0, 0, -1, 0)
	for i := 0; i < segments; i++ {
		vertices = append(vertices, rimPos[i].X(), rimPos[i].Y(), rimPos[i].Z(), 0, -1, 0)
	}

	indices := make([]uint32, 0, segments*6)
	for i := 0; i < segments; i++ {
		curr := uint32(1 + i)
		next := uint32(1 + (i+1)%segments)
		indices = append(indices, 0, next, curr)
	}

	bottomCenter := uint32(segments + 1)
	bottomBase := uint32(segments + 2)
	for i := 0; i < segments; i++ {
		curr := bottomBase + uint32(i)
		next := bottomBase + uint32((i+1)%segments)
		indices = append(indices, bottomCenter, curr, next)
	}

	return createMesh(vertices, indices)
}

func createMesh(vertices []float32, indices []uint32) *Mesh {
	m := &Mesh{
		IndexCount:  int32(len(indices)),
		ModelMatrix: mgl32.Ident4(),
		Color:       mgl32.Vec3{1.0, 1.0, 1.0},
	}

	gl.GenVertexArrays(1, &m.VAO)
	gl.BindVertexArray(m.VAO)

	gl.GenBuffers(1, &m.VBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, m.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.GenBuffers(1, &m.EBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, m.EBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, VertexStride, 0)
	gl.EnableVertexAttribArray(0)

	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, VertexStride, 3*4)
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	return m
}
