package camera

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

type Camera struct {
	Position mgl32.Vec3
	Front    mgl32.Vec3
	Up       mgl32.Vec3
	Right    mgl32.Vec3

	Yaw   float32
	Pitch float32

	FOV       float32
	Aspect    float32
	NearPlane float32
	FarPlane  float32

	MoveSpeed        float32
	MouseSensitivity float32

	PitchClamp float32
	WorldUp    mgl32.Vec3
}

func NewCamera(position mgl32.Vec3, yaw, pitch float32) *Camera {
	c := &Camera{
		Position:         position,
		Front:            mgl32.Vec3{0, 0, -1},
		WorldUp:          mgl32.Vec3{0, 1, 0},
		Yaw:              yaw,
		Pitch:            pitch,
		FOV:              60.0,
		Aspect:           16.0 / 9.0,
		NearPlane:        0.1,
		FarPlane:         500.0,
		MoveSpeed:        15.0,
		MouseSensitivity: 0.1,
		PitchClamp:       89.0,
	}
	c.updateVectors()
	return c
}

func (c *Camera) ViewMatrix() mgl32.Mat4 {
	return mgl32.LookAtV(c.Position, c.Position.Add(c.Front), c.Up)
}

func (c *Camera) ProjectionMatrix() mgl32.Mat4 {
	return mgl32.Perspective(
		mgl32.DegToRad(c.FOV),
		c.Aspect,
		c.NearPlane,
		c.FarPlane,
	)
}

func (c *Camera) ViewProjectionMatrix() mgl32.Mat4 {
	return c.ProjectionMatrix().Mul4(c.ViewMatrix())
}

func (c *Camera) ProcessMouseMovement(xOffset, yOffset float32) {
	xOffset *= c.MouseSensitivity
	yOffset *= c.MouseSensitivity

	c.Yaw += xOffset
	c.Pitch += yOffset

	if c.Pitch > c.PitchClamp {
		c.Pitch = c.PitchClamp
	}
	if c.Pitch < -c.PitchClamp {
		c.Pitch = -c.PitchClamp
	}

	c.updateVectors()
}

func (c *Camera) ProcessKeyboard(direction string, deltaTime float32) {
	velocity := c.MoveSpeed * deltaTime

	switch direction {
	case "forward":
		c.Position = c.Position.Add(c.Front.Mul(velocity))
	case "backward":
		c.Position = c.Position.Sub(c.Front.Mul(velocity))
	case "left":
		c.Position = c.Position.Sub(c.Right.Mul(velocity))
	case "right":
		c.Position = c.Position.Add(c.Right.Mul(velocity))
	case "up":
		c.Position = c.Position.Add(c.WorldUp.Mul(velocity))
	case "down":
		c.Position = c.Position.Sub(c.WorldUp.Mul(velocity))
	}
}

func (c *Camera) updateVectors() {
	yawRad := mgl32.DegToRad(c.Yaw)
	pitchRad := mgl32.DegToRad(c.Pitch)

	front := mgl32.Vec3{
		float32(math.Cos(float64(pitchRad)) * math.Cos(float64(yawRad))),
		float32(math.Sin(float64(pitchRad))),
		float32(math.Cos(float64(pitchRad)) * math.Sin(float64(yawRad))),
	}
	c.Front = front.Normalize()
	c.Right = c.Front.Cross(c.WorldUp).Normalize()
	c.Up = c.Right.Cross(c.Front).Normalize()
}

func (c *Camera) SetAspectRatio(width, height int) {
	c.Aspect = float32(width) / float32(height)
}
