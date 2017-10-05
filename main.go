package main

import (
	"github.com/getlantern/systray"
	"bytes"
	"log"
	"image"
	"github.com/disintegration/imaging"
	"image/color"
	"image/png"
	"net/http"
	"crypto/tls"
	"math/rand"
	"encoding/json"
	"time"
	"github.com/deckarep/gosx-notifier"
)

func main() {
	systray.Run(onReady, onExit)
}

type Data struct {
	Tasks [] struct {
		//TaskStaged   int8 `json:"tasksStaged"`
		//TasksRunning int8 `json:"tasksRunning"`
		//TasksHealthy int8 `json:"tasksHealthy"`
		//Instances    int8 `json:"instances"`
		Id string `json:"id"`
		HealthCheckResults [] struct {
			Alive bool `json:"alive"`
		} `json:"healthCheckResults"`
	} `json:"tasks"`
}

// fill
var login = ""
var password = ""
var link = ""

var light = open("icons/light_slot.png")
var red = open("icons/red_slot.png")
var dark = open("icons/dark_slot.png")
var green = open("icons/green_slot.png")
var yellow = open("icons/yellow_slot.png")

var background = open("icons/triangle.png")

type Node struct {
	Id     string
	Color  image.Image
	X      int
	Y      int
	Rotate int
}

var nodes map[string]*Node = make(map[string]*Node)

var queue = make([][3]int, 24)

var started bool = false
var repaint = make(chan bool, 1)

func notify(app string, title string, message string) {
	note := gosxnotifier.NewNotification(message)
	note.Subtitle = title
	note.Sound = gosxnotifier.Basso
	note.Group = title
	note.Link = link
	note.AppIcon = "triangle.png"
	//note.ContentImage = "gopher.png"
	err := note.Push()
	if err != nil {
		log.Println("Uh oh!")
	}
}

func onReady() {

	apps := [7]string{
		"passport-admin-api",
		"openid-green",
		"cerberus-mini-green",
		"passport-dashboard-green",
		"openid-blue",
		"cerberus-mini-blue",
		"passport-dashboard-blue",
	}

	queue[0] = [3]int{91, 143, 0}
	queue[1] = [3]int{80, 262, 1}
	queue[2] = [3]int{91, 380, 0}
	queue[3] = [3]int{80, 499, 1}
	queue[4] = [3]int{91, 618, 0}

	queue[5] = [3]int{297, 24, 0}
	queue[6] = [3]int{286, 143, 1}
	queue[7] = [3]int{297, 262, 0}
	queue[8] = [3]int{286, 380, 1}
	queue[9] = [3]int{297, 499, 0}
	queue[10] = [3]int{286, 618, 1}
	queue[11] = [3]int{297, 737, 0}

	queue[12] = [3]int{491, 24, 1}
	queue[13] = [3]int{502, 143, 0}
	queue[14] = [3]int{491, 262, 1}
	queue[15] = [3]int{502, 380, 0}
	queue[16] = [3]int{491, 499, 1}
	queue[17] = [3]int{502, 618, 0}
	queue[18] = [3]int{491, 737, 1}

	queue[19] = [3]int{696, 143, 1}
	queue[20] = [3]int{707, 262, 0}
	queue[21] = [3]int{696, 380, 1}
	queue[22] = [3]int{696, 499, 0}
	queue[23] = [3]int{696, 618, 1}

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	systray.SetTooltip("Marathon")

	start := systray.AddMenuItem("Start", "Start application")

	systray.AddSeparator()
	for _, application := range apps {
		item := systray.AddMenuItem(application, "")
		url := link + application + "/restart"
		go func() {
			app := application
			for {
				<-item.ClickedCh
				if !started {
					continue
				}

				req, err := http.NewRequest("POST", url, nil)
				if err != nil {
					log.Fatalf("Request failed: %v", err)
				}
				req.Header.Add("Content-Type", "application/json")
				req.SetBasicAuth(login, password)
				resp, err := client.Do(req)
				if err != nil {
					log.Fatalf("Request failed: %v", err)
				}
				if resp.StatusCode != 200 {
					log.Fatal(resp)
				}
				notify(app, "Marathon", "Restart "+app)
			}
		}()
	}
	systray.AddSeparator()
	go func() {
		for {
			<-start.ClickedCh
			if started {
				start.SetTitle("Start")
			} else {
				start.SetTitle("Stop")
			}
			started = !started
			repaint <- started
		}
	}()

	quit := systray.AddMenuItem("Quit", "Quit the whole app")
	go func() {
		<-quit.ClickedCh
		systray.Quit()
	}()

	//notify := notificator.New(notificator.Options{
	//	DefaultIcon: "/Users/ruslanmikhalev/go/src/github.com/0xAX/notificator/icon/golang.png",
	//	AppName:     "My test App",
	//})

	go func() {
		for {
			buf := new(bytes.Buffer)
			err := png.Encode(buf, GetImage())
			if err != nil {
				log.Fatalf("Encode failed: %v", err)
			}

			systray.SetIcon(buf.Bytes())
			<-repaint
		}
	}()

	go func() {

		for {
			time.Sleep(100 * time.Millisecond)
			if !started {
				continue
			}

			var nodes2 map[string]*Node = make(map[string]*Node)

			for _, app := range apps {

				req, err := http.NewRequest("GET", link+app+"/tasks", nil)
				if err != nil {
					log.Fatalf("Something wrong: %v", err)
				}
				req.SetBasicAuth(login, password)
				resp, err := client.Do(req)

				if err != nil {
					log.Fatalf("Request failed: %v", err)
				}

				var a Data

				if resp.StatusCode != 200 {
					log.Fatalf("Wrong status: %v", resp.Body)
				}

				err = json.NewDecoder(resp.Body).Decode(&a)
				if err != nil {
					log.Fatalf("Decode failed: %v", err)
				}

				for _, task := range a.Tasks {
					c := func() image.Image {
						if len(task.HealthCheckResults) == 0 {
							return yellow
						} else {
							if task.HealthCheckResults[0].Alive == true {
								return green
							} else {
								return red
							}
						}
					}()
					if nodes[task.Id] == nil {
						x := queue[0][0]
						y := queue[0][1]
						nodes2[task.Id] = &Node{Id: task.Id, Color: c, X: x, Y: y, Rotate: queue[0][2]}

						// Discard top element
						queue = queue[1:]
					} else {
						nodes[task.Id].Color = c
						nodes2[task.Id] = nodes[task.Id]
						delete(nodes, task.Id)
					}
				}
				//notify.Push("Marathon", fmt.Sprint(a), "", notificator.UR_NORMAL)
			}
			for _, node := range nodes {
				queue = append(queue, [3]int{node.X, node.Y, node.Rotate})
			}
			nodes = nodes2

			repaint <- true
		}

	}()
}

func open(path string) image.Image {
	var im, err = imaging.Open(path)
	if err != nil {
		log.Fatalf("Open failed: %v", err)
	}
	return im
}

func Shuffle(a [][3]int) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func GetImage() image.Image {

	Shuffle(queue)

	//var randIm image.Image = func() image.Image {
	//	if rand.Int()%2 == 0 {
	//		return light
	//	} else {
	//		return red
	//	}
	//}()
	//
	var canvas *image.NRGBA = imaging.New(960, 960, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if !started {
		canvas = imaging.Overlay(canvas, background, image.Pt(0, 0), 1.0)
		canvas = imaging.Grayscale(canvas)
		return canvas
	}
	canvas = imaging.Overlay(canvas, background, image.Pt(0, 0), 0.7)

	for _, node := range nodes {
		var i = func() image.Image {
			if node.Rotate == 1 {
				return imaging.Rotate180(node.Color)
			} else {
				return node.Color
			}
		}()

		canvas = imaging.Overlay(canvas, i, image.Pt(node.X, node.Y), 1)
	}
	//
	//canvas = imaging.Overlay(canvas, light, image.Pt(91, 143), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(80, 262), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(91, 380), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(80, 499), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(91, 618), 1)
	//
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 24), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(286, 143), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 262), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(randIm), image.Pt(286, 380), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 499), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(286, 618), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 737), 1)

	//canvas = imaging.Overlay(canvas, imaging.Rotate180(randIm), image.Pt(491, 24), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(502, 143), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(randIm), image.Pt(491, 262), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(502, 380), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(491, 499), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(502, 618), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(491, 737), 1)

	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(696, 143), 1)
	//canvas = imaging.Overlay(canvas, yellow, image.Pt(707, 262), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(696, 380), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(696, 499), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(696, 618), 1)
	//
	// //PUZZLE
	//canvas = imaging.Overlay(canvas, light, image.Pt(91, 143), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(80, 262), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(91, 380), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(80, 499), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(91, 618), 1)
	//
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 24), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(286, 143), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(297, 262), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(286, 380), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 499), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(286, 618), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(297, 737), 1)
	//
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(491, 24), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(502, 143), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(491, 262), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(502, 380), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(491, 499), 1)
	//canvas = imaging.Overlay(canvas, dark, image.Pt(502, 618), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(dark), image.Pt(491, 737), 1)
	//
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(696, 143), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(707, 262), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(696, 380), 1)
	//canvas = imaging.Overlay(canvas, light, image.Pt(696, 499), 1)
	//canvas = imaging.Overlay(canvas, imaging.Rotate180(light), image.Pt(696, 618), 1)

	return canvas
}

func onExit() {

}
