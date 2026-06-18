package scene

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Terrain struct {
	Mesh      Mesh
	WorldSize float32
	MaxHeight float32

	worldVertices []mgl32.Vec3
}

func NewTerrain(worldSize, maxHeight float32) *Terrain {
	return &Terrain{
		WorldSize: worldSize,
		MaxHeight: maxHeight,
	}
}

func (t *Terrain) LoadOBJGeometry(path string) error {
	mesh, err := LoadOBJ(path)
	if err != nil {
		return err
	}

	bbMin, bbMax, rawPositions := t.readVBOData(mesh.VBO)
	bbSize := bbMax.Sub(bbMin)
	if bbSize.X() <= 0 || bbSize.Z() <= 0 || bbSize.Y() <= 0 {
		return fmt.Errorf("вырожденный bounding box у %s: size=%.2f,%.2f,%.2f", path, bbSize.X(), bbSize.Y(), bbSize.Z())
	}

	scaleXZ := t.WorldSize / float32(math.Max(float64(bbSize.X()), float64(bbSize.Z())))
	scaleY := t.MaxHeight / bbSize.Y()

	centerXZ := (bbMin.X() + bbMax.X()) * 0.5
	centerZZ := (bbMin.Z() + bbMax.Z()) * 0.5

	t.Mesh = *mesh
	t.Mesh.ModelMatrix = mgl32.Ident4().
		Mul4(mgl32.Translate3D(-centerXZ, -bbMin.Y(), -centerZZ)).
		Mul4(mgl32.Scale3D(scaleXZ, scaleY, scaleXZ))

	t.worldVertices = make([]mgl32.Vec3, len(rawPositions))
	for i, p := range rawPositions {

		wp := mgl32.Vec3{
			(p.X() - centerXZ) * scaleXZ,
			(p.Y() - bbMin.Y()) * scaleY,
			(p.Z() - centerZZ) * scaleXZ,
		}
		t.worldVertices[i] = wp
	}

	return nil
}

func (t *Terrain) readVBOData(vbo uint32) (mgl32.Vec3, mgl32.Vec3, []mgl32.Vec3) {
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	var bufSize int32
	gl.GetBufferParameteriv(gl.ARRAY_BUFFER, gl.BUFFER_SIZE, &bufSize)
	if bufSize == 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		return mgl32.Vec3{}, mgl32.Vec3{}, nil
	}

	data := make([]float32, bufSize/4)
	gl.GetBufferSubData(gl.ARRAY_BUFFER, 0, int(bufSize), gl.Ptr(data))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	vertexCount := len(data) / 6
	if vertexCount == 0 {
		return mgl32.Vec3{}, mgl32.Vec3{}, nil
	}

	bbMin := mgl32.Vec3{data[0], data[1], data[2]}
	bbMax := bbMin
	positions := make([]mgl32.Vec3, vertexCount)

	for i := 0; i < vertexCount; i++ {
		off := i * 6
		x, y, z := data[off], data[off+1], data[off+2]
		positions[i] = mgl32.Vec3{x, y, z}

		bbMin = mgl32.Vec3{
			float32(math.Min(float64(bbMin.X()), float64(x))),
			float32(math.Min(float64(bbMin.Y()), float64(y))),
			float32(math.Min(float64(bbMin.Z()), float64(z))),
		}
		bbMax = mgl32.Vec3{
			float32(math.Max(float64(bbMax.X()), float64(x))),
			float32(math.Max(float64(bbMax.Y()), float64(y))),
			float32(math.Max(float64(bbMax.Z()), float64(z))),
		}
	}

	return bbMin, bbMax, positions
}

func (t *Terrain) SampleHeight(wx, wz float32) float32 {
	if len(t.worldVertices) == 0 {
		return 0
	}

	bestDist := float32(math.MaxFloat32)
	bestY := float32(0.0)

	for _, v := range t.worldVertices {
		dx := v.X() - wx
		dz := v.Z() - wz
		dist := dx*dx + dz*dz
		if dist < bestDist {
			bestDist = dist
			bestY = v.Y()
		}
	}

	return bestY
}

func (t *Terrain) Draw() {
	t.Mesh.Draw()
}

func (t *Terrain) Release() {
	t.Mesh.Release()
}
