package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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

	tria2 := make(chan Triangle)
	vertices := []Triangle{}
	vertices = append(vertices, points)
	vaos := makeVao(vertices)
	newObject := false

	draw(vaos, window, program)
	go func() {
		for range ticker2.C {
			readMessage(c, tria2)
		}
	}()
	go func() {
		for range ticker.C {
			sendJSON(c, points)
		}
	}()
	go func() {
		for tri := range tria2 {
			dup := false
			for _, v := range vertices {
				if Equal(v, tri) {
					dup = true
				}
			}
			if !dup && len(tri) > 0 {
				vertices = append(vertices, tri)
				newObject = true
			}
		}
	}()

	for !window.ShouldClose() {
		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
		}
		if newObject {
			vaos = makeVao(vertices)
			newObject = false
		}
		draw(vaos, window, program)
	}
	defer c.Close()
}

func readMessage(c *websocket.Conn, trian2 chan Triangle) {
	triangle := Triangle{}
	err := c.ReadJSON(&triangle)
	if err != nil {
		log.Fatalln("read:", err)
		return
	}
	log.Printf("recv: %f", triangle)
	if len(triangle) > 0 {
		trian2 <- triangle
	}
}

func sendJSON(c *websocket.Conn, points Triangle) {
	err := c.WriteJSON(points)
	if err != nil {
		log.Fatalln("write:", err)
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
func Equal(a, b Triangle) bool {
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
	0, 0, 0, //0, 0, 0,
	0, 0, 0, //0, 0, 0,
	0, 0, 0, //0, 0, 0,
}

func randTriangle() Triangle {

	rand.Seed(time.Now().UnixNano())
	vertexTop1, veretxTop2 := randomizeVertex(), randomizeVertex()
	vertextLeft1, veretxLeft2 := randomizeVertex(), randomizeVertex()
	vertextRight1, veretxRight2 := randomizeVertex(), randomizeVertex()
	shape := Triangle{
		vertexTop1, veretxTop2, 0, 1, 0, 0,
		vertextLeft1, veretxLeft2, 0, 0, 1, 0,
		vertextRight1, veretxRight2, 0, 0, 0, 1,
	}
	return shape
}

func randTriangleColor() float32 {
	rand.Seed(time.Now().UnixNano())
	return rand.Float32()
}

func normalize(val int, max float32, min float32) float32 {
	return (float32(val) - min) / (max - min)
}

// makeVao initializes and returns a vertex array from the points provided.
func makeVao(vertices []Triangle) []uint32 {
	var vaos []uint32
	var vbo uint32
	for _, vert := range vertices {
		var vertice []float32
		for _, v := range vert {
			vertice = append(vert, v)
		}
		log.Println(vertice)
		gl.GenBuffers(1, &vbo)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
		gl.BufferData(gl.ARRAY_BUFFER, 8*len(vertice), gl.Ptr(vertice), gl.DYNAMIC_DRAW)

		var vao uint32
		gl.GenVertexArrays(1, &vao)
		gl.BindVertexArray(vao)
		// gl.EnableVertexAttribArray(0)
		gl.VertexAttribPointer(0, 3, gl.FLOAT, false,
			6*4, gl.PtrOffset(0))
		gl.EnableVertexAttribArray(0)

		gl.VertexAttribPointer(1, 3, gl.FLOAT, false,
			6*4, gl.PtrOffset(3*4))
		gl.EnableVertexAttribArray(1)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
		// gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

		vaos = append(vaos, vao)
		gl.BindVertexArray(0)
	}

	return vaos
}

func draw(vaos []uint32, window *glfw.Window, program uint32) {

	gl.ClearColor(0.1, 0.1, 0.1, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	for _, vao := range vaos {
		gl.BindVertexArray(vao)

		// len(triangle) 9 vertices / 3
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(triangleVertexArray)/3))
		gl.BindVertexArray(0)
	}

	// my lasers
	// color := normalize(time.Now().Nanosecond(), 999999999, 0)
	// myColor := float32(math.Asin(float64(color)))
	// myRevColor := float32(math.Cos((float64(color))))
	// triangleColor := gl.GetUniformLocation(program, gl.Str("triangleColor\x00"))
	// gl.Uniform3f(triangleColor, myColor, myRevColor, myRevColor)

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

	glfw.WindowHint(glfw.Samples, 8)

	x := glfw.GetPrimaryMonitor().GetVideoMode().Width
	y := glfw.GetPrimaryMonitor().GetVideoMode().Height
	x = 320
	y = 240
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

	vertexShaderSource, err := ioutil.ReadFile("vertexShader.glsl")
	if err != nil {
		panic(err)
	}
	vertexShader, err := compileShader(string(vertexShaderSource)+"\x00", gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShaderSource, err := ioutil.ReadFile("fragmentShader.glsl")
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(string(fragmentShaderSource)+"\x00", gl.FRAGMENT_SHADER)
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
