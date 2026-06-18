package main

import (
	"fmt"
	"log"
	"math"
	"runtime"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/cherevatovm/graphics-csm-project/pkg/camera"
	"github.com/cherevatovm/graphics-csm-project/pkg/renderer"
	"github.com/cherevatovm/graphics-csm-project/pkg/scene"
	"github.com/cherevatovm/graphics-csm-project/pkg/shader"
	"github.com/cherevatovm/graphics-csm-project/pkg/shadow"
)

const (
	WindowWidth  = 1280
	WindowHeight = 720
	WindowTitle  = "Cascaded Shadow Maps"
)

const (
	CascadeCount        = 4
	ShadowMapResolution = 4096
	CascadeLambda       = 0.75
	CameraNear          = 0.1
	CameraFar           = 400.0
)

var (
	window       *glfw.Window
	cam          *camera.Camera
	sunAzimuth   float32 = 45.0
	sunElevation float32 = 35.0

	debugMode bool

	lastFrameTime float64
	deltaTime     float32

	firstMouse bool = true
	lastMouseX float64
	lastMouseY float64

	fboManager   *shadow.FBOManager
	cascadeCalc  *shadow.CascadeCalculator
	shadowRend   *renderer.ShadowRenderer
	mainRend     *renderer.MainRenderer
	debugOverlay *renderer.DebugOverlay
	testScene    *scene.Scene
)

func main() {
	runtime.LockOSThread()
	if err := initWindow(); err != nil {
		log.Fatalf("Ошибка инициализации окна: %v", err)
	}
	defer glfw.Terminate()
	if err := initOpenGL(); err != nil {
		log.Fatalf("Ошибка инициализации OpenGL: %v", err)
	}

	window.SetFramebufferSizeCallback(framebufferSizeCallback)
	window.SetCursorPosCallback(mouseCallback)
	window.SetKeyCallback(keyCallback)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	shadowDepthProgram, err := shader.LoadFromFiles(
		"assets/shaders/shadow_depth.vert",
		"assets/shaders/shadow_depth.frag",
	)
	if err != nil {
		log.Fatalf("Ошибка загрузки shadow_depth шейдера: %v", err)
	}
	defer shadowDepthProgram.Delete()

	mainSceneProgram, err := shader.LoadFromFiles(
		"assets/shaders/main_scene.vert",
		"assets/shaders/main_scene.frag",
	)
	if err != nil {
		log.Fatalf("Ошибка загрузки main_scene шейдера: %v", err)
	}
	defer mainSceneProgram.Delete()

	debugLinesProgram, err := shader.LoadFromFiles(
		"assets/shaders/debug_lines.vert",
		"assets/shaders/debug_lines.frag",
	)
	if err != nil {
		log.Fatalf("Ошибка загрузки debug_lines шейдера: %v", err)
	}
	defer debugLinesProgram.Delete()

	fboManager, err = shadow.NewFBOManager(CascadeCount, ShadowMapResolution)
	if err != nil {
		log.Fatalf("Ошибка создания FBO: %v", err)
	}
	defer fboManager.Release()

	cascadeCalc = shadow.NewCascadeCalculator(shadow.CascadeConfig{
		Count:         CascadeCount,
		Lambda:        CascadeLambda,
		CameraNear:    CameraNear,
		CameraFar:     CameraFar,
		ShadowMapSize: ShadowMapResolution,
	})

	shadowRend = renderer.NewShadowRenderer(fboManager, cascadeCalc, shadowDepthProgram)
	mainRend = renderer.NewMainRenderer(fboManager, cascadeCalc, mainSceneProgram)
	debugOverlay = renderer.NewDebugOverlay(debugLinesProgram)

	testScene = createTestScene()
	defer testScene.ReleaseAll()

	cam = camera.NewCamera(
		mgl32.Vec3{0, 15, 60},
		-90.0,
		-15.0,
	)
	cam.SetAspectRatio(WindowWidth, WindowHeight)
	cam.MoveSpeed = 20.0

	fmt.Println("\n=== Управление ===")
	fmt.Println("  WASD       - перемещение камеры")
	fmt.Println("  Мышь       - вращение камеры")
	fmt.Println("  Стрелки    - вращение солнца")
	fmt.Println("  F1         - дебаг-режим (цвета каскадов)")
	fmt.Println("  ESC        - выход")
	fmt.Println("==================")

	lastFrameTime = glfw.GetTime()

	for !window.ShouldClose() {
		currentTime := glfw.GetTime()
		deltaTime = float32(currentTime - lastFrameTime)
		lastFrameTime = currentTime

		processContinuousInput()

		sunDir := sunDirection()
		fbWidth, fbHeight := window.GetFramebufferSize()
		shadowRend.RenderPass(testScene, cam, sunDir)

		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		mainRend.Render(testScene, cam, sunDir, debugMode)
		if debugMode {
			debugOverlay.DrawFrustums(cascadeCalc, cam, sunDir)
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func initWindow() error {
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("ошибка инициализации GLFW: %w", err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	glfw.WindowHint(glfw.Samples, 4)

	var err error
	window, err = glfw.CreateWindow(WindowWidth, WindowHeight, WindowTitle, nil, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания окна: %w", err)
	}

	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	return nil
}

func initOpenGL() error {
	if err := gl.Init(); err != nil {
		return fmt.Errorf("ошибка инициализации OpenGL: %w", err)
	}

	fmt.Printf("OpenGL версия: %s\n", gl.GoStr(gl.GetString(gl.VERSION)))
	fmt.Printf("GPU: %s\n", gl.GoStr(gl.GetString(gl.RENDERER)))
	fmt.Printf("GLSL версия: %s\n", gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION)))

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.Enable(gl.MULTISAMPLE)
	gl.ClearColor(0.1, 0.15, 0.25, 1.0)

	return nil
}

func createTestScene() *scene.Scene {
	s := scene.NewScene()

	terrain := scene.NewTerrain(255.0, 16.0)
	if err := terrain.LoadOBJGeometry("assets/3d_models/terrain.obj"); err != nil {
		log.Fatalf("Ошибка загрузки terrain.obj: %v", err)
	}
	s.Terrain = terrain

	pineMesh, err := scene.LoadOBJ("assets/3d_models/pinetree.obj")
	if err != nil {
		fmt.Printf("Не удалось загрузить pinetree.obj: %v\n", err)
		fmt.Println("Сцена будет только с ландшафтом.")
		return s
	}

	teapotMesh, err := scene.LoadOBJ("assets/3d_models/utah_teapot_lowpoly.obj")
	if err != nil {
		fmt.Printf("Не удалось загрузить чайник: %v\n", err)
	}

	type treePlacement struct {
		x, z    float32
		scale   float32
		rotYDeg float32
	}
	pinePlacements := []treePlacement{
		{-80, -72, 0.12, 15},
		{-40, -60, 0.10, 45},
		{-20, -70, 0.09, 80},
		{10, -65, 0.11, -20},
		{50, -75, 0.08, 110},
		{70, -60, 0.13, -55},

		{-82, -25, 0.11, -10},
		{-50, -15, 0.09, 70},
		{-10, -28, 0.12, 35},
		{20, -18, 0.08, -70},
		{40, -30, 0.10, 5},
		{80, -20, 0.14, 60},

		{-68, 20, 0.09, 40},
		{-42, 30, 0.13, -35},
		{-18, 18, 0.08, 90},
		{12, 28, 0.10, -45},
		{52, 15, 0.11, 25},
		{72, 32, 0.07, -80},

		{-78, 60, 0.10, -25},
		{-38, 72, 0.12, 55},
		{-22, 65, 0.09, 10},
		{8, 72, 0.11, -60},
		{48, 58, 0.08, 75},
		{78, 68, 0.13, -15},
	}

	greenColor := mgl32.Vec3{0.15, 0.55, 0.18}
	for _, tp := range pinePlacements {
		groundY := terrain.SampleHeight(tp.x, tp.z)

		yOffset := groundY + 16.0*tp.scale + 0.1
		modelMat := mgl32.Ident4().
			Mul4(mgl32.Translate3D(tp.x, yOffset+5.0, tp.z)).
			Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(tp.rotYDeg))).
			Mul4(mgl32.Scale3D(tp.scale, tp.scale, tp.scale))

		pineInstance := &scene.Mesh{
			VAO:         pineMesh.VAO,
			VBO:         pineMesh.VBO,
			EBO:         pineMesh.EBO,
			IndexCount:  pineMesh.IndexCount,
			ModelMatrix: modelMat,
			Color:       greenColor,
		}
		s.AddMesh(pineInstance)
	}

	if teapotMesh != nil {
		redColor := mgl32.Vec3{0.7, 0.12, 0.08}
		teapotPlacements := []treePlacement{
			{-65, -55, 4.5, 10},
			{-15, -65, 5.0, 50},
			{25, -58, 4.0, 80},
			{55, -62, 5.5, -30},

			{-58, -5, 4.8, 35},
			{-22, 8, 3.8, -55},
			{18, -3, 5.2, 70},
			{62, 5, 4.2, -20},

			{-62, 55, 4.3, -45},
			{-18, 62, 5.3, 15},
			{22, 58, 3.9, -70},
			{58, 65, 4.7, 40},
		}

		for _, tp := range teapotPlacements {
			groundY := terrain.SampleHeight(tp.x, tp.z)

			modelMat := mgl32.Ident4().
				Mul4(mgl32.Translate3D(tp.x, groundY+3, tp.z)).
				Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(tp.rotYDeg))).
				Mul4(mgl32.Scale3D(tp.scale, tp.scale, tp.scale))

			teapotInstance := &scene.Mesh{
				VAO:         teapotMesh.VAO,
				VBO:         teapotMesh.VBO,
				EBO:         teapotMesh.EBO,
				IndexCount:  teapotMesh.IndexCount,
				ModelMatrix: modelMat,
				Color:       redColor,
			}
			s.AddMesh(teapotInstance)
		}
	}

	return s
}

func processContinuousInput() {
	if window.GetKey(glfw.KeyW) == glfw.Press {
		cam.ProcessKeyboard("forward", deltaTime)
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		cam.ProcessKeyboard("backward", deltaTime)
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		cam.ProcessKeyboard("left", deltaTime)
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		cam.ProcessKeyboard("right", deltaTime)
	}
	if window.GetKey(glfw.KeySpace) == glfw.Press {
		cam.ProcessKeyboard("up", deltaTime)
	}
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		cam.ProcessKeyboard("down", deltaTime)
	}

	const sunSpeed = 45.0
	if window.GetKey(glfw.KeyLeft) == glfw.Press {
		sunAzimuth -= sunSpeed * deltaTime
	}
	if window.GetKey(glfw.KeyRight) == glfw.Press {
		sunAzimuth += sunSpeed * deltaTime
	}
	if window.GetKey(glfw.KeyUp) == glfw.Press {
		sunElevation += sunSpeed * deltaTime
		if sunElevation > 89.0 {
			sunElevation = 89.0
		}
	}
	if window.GetKey(glfw.KeyDown) == glfw.Press {
		sunElevation -= sunSpeed * deltaTime
		if sunElevation < 20.0 {
			sunElevation = 20.0
		}
	}

	for sunAzimuth < 0 {
		sunAzimuth += 360
	}
	for sunAzimuth >= 360 {
		sunAzimuth -= 360
	}
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyF1:
			debugMode = debugOverlay.Toggle()
			if debugMode {
				fmt.Println("Дебаг-режим: ВКЛ")
			} else {
				fmt.Println("Дебаг-режим: ВЫКЛ")
			}
		}
	}
}

func mouseCallback(w *glfw.Window, xpos, ypos float64) {
	if firstMouse {
		lastMouseX = xpos
		lastMouseY = ypos
		firstMouse = false
	}

	xOffset := float32(xpos - lastMouseX)
	yOffset := float32(lastMouseY - ypos)

	lastMouseX = xpos
	lastMouseY = ypos

	cam.ProcessMouseMovement(xOffset, yOffset)
}

func framebufferSizeCallback(w *glfw.Window, width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
	cam.SetAspectRatio(width, height)
}

func sunDirection() mgl32.Vec3 {
	azRad := mgl32.DegToRad(sunAzimuth)
	elRad := mgl32.DegToRad(sunElevation)

	return mgl32.Vec3{
		float32(math.Cos(float64(elRad)) * math.Sin(float64(azRad))),
		float32(-math.Sin(float64(elRad))),
		float32(math.Cos(float64(elRad)) * math.Cos(float64(azRad))),
	}.Normalize()
}
