package shadow

import (
	"math"

	"github.com/cherevatovm/graphics-csm-project/pkg/camera"
	"github.com/go-gl/mathgl/mgl32"
)

type Cascade struct {
	FarPlane      float32
	LightViewProj mgl32.Mat4
}

type CascadeConfig struct {
	Count         int
	Lambda        float32
	CameraNear    float32
	CameraFar     float32
	ShadowMapSize float32
}

type CascadeCalculator struct {
	Config   CascadeConfig
	Cascades []Cascade
}

func NewCascadeCalculator(config CascadeConfig) *CascadeCalculator {
	return &CascadeCalculator{
		Config:   config,
		Cascades: make([]Cascade, config.Count),
	}
}

func (cc *CascadeCalculator) Calculate(cam *camera.Camera, sunDir mgl32.Vec3) {
	sunDir = sunDir.Normalize()
	farPlanes := cc.calculateSplitDistances()
	near := cc.Config.CameraNear

	for i := 0; i < cc.Config.Count; i++ {
		far := farPlanes[i]
		corners := cc.GetFrustumCorners(near, far, cam)
		lvp := cc.buildLightViewProj(corners, sunDir)
		cc.Cascades[i] = Cascade{
			FarPlane:      far,
			LightViewProj: lvp,
		}
		near = far
	}
}

func (cc *CascadeCalculator) calculateSplitDistances() []float32 {
	n := cc.Config.CameraNear
	f := cc.Config.CameraFar
	lambda := cc.Config.Lambda
	count := float32(cc.Config.Count)
	distances := make([]float32, cc.Config.Count)

	for i := 1; i <= cc.Config.Count; i++ {
		ratio := float32(i) / count
		logPart := n * float32(math.Pow(float64(f/n), float64(ratio)))
		uniformPart := n + (f-n)*ratio
		distances[i-1] = lambda*logPart + (1-lambda)*uniformPart
	}

	return distances
}

func (cc *CascadeCalculator) GetFrustumCorners(near, far float32, cam *camera.Camera) [8]mgl32.Vec3 {
	proj := cam.ProjectionMatrix()
	view := cam.ViewMatrix()
	invVP := proj.Mul4(view).Inv()

	camNear := cam.NearPlane
	camFar := cam.FarPlane
	fpn := camFar + camNear
	fmn := camFar - camNear
	twoFN := 2.0 * camFar * camNear

	ndcNear := fpn/fmn + twoFN/(fmn*(-near))
	ndcFar := fpn/fmn + twoFN/(fmn*(-far))

	ndcCorners := [8]mgl32.Vec4{
		{-1, -1, ndcNear, 1},
		{1, -1, ndcNear, 1},
		{1, 1, ndcNear, 1},
		{-1, 1, ndcNear, 1},

		{-1, -1, ndcFar, 1},
		{1, -1, ndcFar, 1},
		{1, 1, ndcFar, 1},
		{-1, 1, ndcFar, 1},
	}

	var worldCorners [8]mgl32.Vec3
	for i := 0; i < 8; i++ {
		worldHomog := invVP.Mul4x1(ndcCorners[i])

		w := worldHomog.W()
		worldCorners[i] = mgl32.Vec3{
			worldHomog.X() / w,
			worldHomog.Y() / w,
			worldHomog.Z() / w,
		}
	}

	return worldCorners
}

func (cc *CascadeCalculator) buildLightViewProj(corners [8]mgl32.Vec3, lightDir mgl32.Vec3) mgl32.Mat4 {
	center := mgl32.Vec3{}
	for i := 0; i < 8; i++ {
		center = center.Add(corners[i])
	}
	center = center.Mul(1.0 / 8.0)

	radius := float32(0.0)
	for i := 0; i < 8; i++ {
		dist := corners[i].Sub(center).Len()
		if dist > radius {
			radius = dist
		}
	}

	lightPos := center.Sub(lightDir.Mul(radius))
	up := mgl32.Vec3{0, 1, 0}
	if mgl32.Abs(lightDir.Dot(up)) > 0.999 {
		up = mgl32.Vec3{1, 0, 0}
	}

	lightView := mgl32.LookAtV(lightPos, center, up)
	lightProj := mgl32.Ortho(
		-radius, radius,
		-radius, radius,
		0.0, radius*2.0,
	)

	lvp := lightProj.Mul4(lightView)
	lvp = cc.texelSnap(lvp)

	return lvp
}

func (cc *CascadeCalculator) texelSnap(lvp mgl32.Mat4) mgl32.Mat4 {
	origin := lvp.Mul4x1(mgl32.Vec4{0, 0, 0, 1})

	texelSize := 1.0 / cc.Config.ShadowMapSize
	originX := (origin.X()/origin.W())*0.5 + 0.5
	originY := (origin.Y()/origin.W())*0.5 + 0.5

	snapX := float32(math.Round(float64(originX/texelSize))) * texelSize
	snapY := float32(math.Round(float64(originY/texelSize))) * texelSize

	deltaX := snapX - originX
	deltaY := snapY - originY

	lvp[12] += deltaX * 2.0
	lvp[13] += deltaY * 2.0

	return lvp
}

func (cc *CascadeCalculator) GetCascadeIndex(viewDepth float32) int {
	for i := 0; i < cc.Config.Count; i++ {
		if viewDepth < cc.Cascades[i].FarPlane {
			return i
		}
	}
	return cc.Config.Count - 1
}

func GetCascadeColors() [4]mgl32.Vec3 {
	return [4]mgl32.Vec3{
		{1.0, 0.2, 0.2},
		{0.2, 1.0, 0.2},
		{0.2, 0.4, 1.0},
		{1.0, 1.0, 0.2},
	}
}
