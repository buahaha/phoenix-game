package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	done := make(chan struct{})

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	ticker2 := time.NewTicker(100 * time.Millisecond)
	defer ticker2.Stop()

	runtime.LockOSThread()
	window := initGlfw()
	defer glfw.Terminate()
	program := initOpenGL()

	points := randTriangle()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	tria2 := make(chan []float32)
	vao := makeVao(randTriangle())
	vaos := []uint32{}
	vertices := [][]float32{}
	for !window.ShouldClose() {
		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
		}
		go draw(vaos, window, program)
		go func() {
			select {
			case _ = <-ticker.C:
				go sendJSON(done, c, points)
			}
		}()
		go func() {
			select {
			case _ = <-ticker2.C:
				go readMessage(done, c, tria2)
			}
		}()
		var tri []float32
		tri = <-tria2
		dup := false
		for _, v := range vertices {
			if Equal(v, tri) {
				dup = true
			}
		}
		if !dup {
			vertices = append(vertices, tri)
			vao = makeVao(tri)
			vaos = append(vaos, vao)
		}

	}
	c.Close()
}

func readMessage(done chan struct{}, c *websocket.Conn, trian2 chan []float32) {
	_, tri, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}
	log.Printf("recv: %s", tri)
	triangle := []float32{}
	json.Unmarshal(tri, &triangle)
	if len(triangle) > 0 {
		trian2 <- triangle
	}
}

func sendJSON(done chan struct{}, c *websocket.Conn, points Triangle) {
	err := c.WriteJSON(points)
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func randomizeVertex() float32 {
	var vertex float32
	if rand.Intn(2) > 0 {
		vertex = rand.Float32()
	} else {
		vertex = -rand.Float32()
	}
	return vertex
}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

type Triangle []float32

var triangleVertexArray []float32 = []float32{
	0, 0, 0,
	0, 0, 0,
	0, 0, 0,
}

func randTriangle() Triangle {

	rand.Seed(time.Now().UnixNano())
	vertexTop1, veretxTop2 := randomizeVertex(), randomizeVertex()
	vertextLeft1, veretxLeft2 := randomizeVertex(), randomizeVertex()
	vertextRight1, veretxRight2 := randomizeVertex(), randomizeVertex()
	shape := Triangle{
		vertexTop1, veretxTop2, 0,
		vertextLeft1, veretxLeft2, 0,
		vertextRight1, veretxRight2, 0,
	}
	return shape
}

var vertexShaderSource = `
    #version 410
    in vec3 vp;
    void main() {
        gl_Position = vec4(vp, 1.0);
    }
` + "\x00"

var fragmentShaderSource = `
	#version 410
	uniform vec3 triangleColor;
    out vec4 frag_colour;
    void main() {
        frag_colour = vec4(triangleColor, 1);
    }
` + "\x00"

func randTriangleColor() float32 {
	rand.Seed(time.Now().UnixNano())
	return rand.Float32()
}

// makeVao initializes and returns a vertex array from the points provided.
func makeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 8*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

func draw(vaos []uint32, window *glfw.Window, program uint32) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	fmt.Println("vaos:", len(vaos))
	for _, vao := range vaos {
		gl.BindVertexArray(vao)
		// len(triangle) 9 vertices / 3
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(triangleVertexArray)/3))
	}

	triangleColor := gl.GetUniformLocation(program, gl.Str("triangleColor\x00"))
	gl.Uniform3f(triangleColor, randTriangleColor(), randTriangleColor(), randTriangleColor())

	glfw.PollEvents()
	window.SwapBuffers()
}

// initGlfw initializes glfw and returns a Window to use.
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4) // OR 2
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	x := glfw.GetPrimaryMonitor().GetVideoMode().Width
	y := glfw.GetPrimaryMonitor().GetVideoMode().Height
	x = 640
	y = 480
	window, err := glfw.CreateWindow(x, y, "Phoenix", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

// initOpenGL initializes OpenGL and returns an intiialized program.
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)

	return prog
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}
