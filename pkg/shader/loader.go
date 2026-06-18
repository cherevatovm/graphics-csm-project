package shader

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Program uint32

func LoadFromFiles(vertPath, fragPath string) (Program, error) {
	vertSrc, err := os.ReadFile(vertPath)
	if err != nil {
		return 0, fmt.Errorf("чтение вершинного шейдера %s: %w", vertPath, err)
	}

	fragSrc, err := os.ReadFile(fragPath)
	if err != nil {
		return 0, fmt.Errorf("чтение фрагментного шейдера %s: %w", fragPath, err)
	}

	return Compile(string(vertSrc), string(fragSrc))
}

func Compile(vertSource, fragSource string) (Program, error) {

	vertShader, err := compileShader(gl.VERTEX_SHADER, vertSource)
	if err != nil {
		return 0, fmt.Errorf("вершинный шейдер: %w", err)
	}
	defer gl.DeleteShader(vertShader)

	fragShader, err := compileShader(gl.FRAGMENT_SHADER, fragSource)
	if err != nil {
		return 0, fmt.Errorf("фрагментный шейдер: %w", err)
	}
	defer gl.DeleteShader(fragShader)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertShader)
	gl.AttachShader(prog, fragShader)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetProgramInfoLog(prog, logLen, nil, gl.Str(log))
		gl.DeleteProgram(prog)
		return 0, fmt.Errorf("ошибка линковки программы: %s", log)
	}

	return Program(prog), nil
}

func compileShader(shaderType uint32, source string) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csource, free := gl.Strs(source + "\x00")
	defer free()

	gl.ShaderSource(shader, 1, csource, nil)
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(log))

		shaderTypeName := "VERTEX"
		if shaderType == gl.FRAGMENT_SHADER {
			shaderTypeName = "FRAGMENT"
		}

		gl.DeleteShader(shader)
		return 0, fmt.Errorf("ошибка компиляции %s шейдера:\n%s\n--- Исходный код ---\n%s",
			shaderTypeName, log, source)
	}

	return shader, nil
}

func (p Program) Use() {
	gl.UseProgram(uint32(p))
}

func (p Program) Delete() {
	gl.DeleteProgram(uint32(p))
}

func (p Program) UniformLocation(name string) int32 {
	return gl.GetUniformLocation(uint32(p), gl.Str(name+"\x00"))
}

func (p Program) SetInt(name string, value int32) {
	gl.Uniform1i(p.UniformLocation(name), value)
}

func (p Program) SetFloat(name string, value float32) {
	gl.Uniform1f(p.UniformLocation(name), value)
}

func (p Program) SetVec3(name string, x, y, z float32) {
	gl.Uniform3f(p.UniformLocation(name), x, y, z)
}

func (p Program) SetMat4(name string, mat *mgl32.Mat4) {
	gl.UniformMatrix4fv(p.UniformLocation(name), 1, false, &mat[0])
}

func (p Program) SetBool(name string, value bool) {
	var v int32
	if value {
		v = 1
	}
	gl.Uniform1i(p.UniformLocation(name), v)
}
