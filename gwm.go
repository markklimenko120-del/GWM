package main

import (
	"gwm/config"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"sync"
 	"github.com/jezek/xgb"
	// "github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/xevent"
	"github.com/jezek/xgbutil/xgraphics"
	"github.com/nfnt/resize"
)

var wg sync.WaitGroup 
var keycode xproto.Keycode
var cfg *config.Config
var Errors = make(chan error,10)

type ConnInfo struct {
	Conn *xgb.Conn
	XConn *xgbutil.XUtil
	Setup *xproto.SetupInfo
	Screen xproto.ScreenInfo
}


func CreatePixelMap(CI *ConnInfo,logfile *os.File) (xproto.Pixmap) {
	background,err := xproto.NewPixmapId(CI.Conn)
	if err != nil {
		logfile.WriteString(err.Error())
		Errors <- err
		return background
	}
	xproto.CreatePixmap(CI.Conn, CI.Screen.RootDepth, background, xproto.Drawable(CI.Screen.Root),CI.Screen.WidthInPixels,CI.Screen.HeightInPixels)
	return background
}

func CreateGCtx(CI *ConnInfo,logfile *os.File) xproto.Gcontext {
	gc,err := xproto.NewGcontextId(CI.Conn)
	if err != nil {
		Errors <- err
		logfile.WriteString(err.Error())
		return gc
	}
	xproto.CreateGC(CI.Conn,gc,xproto.Drawable(CI.Screen.Root),0,[]uint32{})
	return gc
}

func GetBG(path string,logfile *os.File) image.Image {
	file,err := os.Open(path)
	if err != nil {
		Errors <- err
		return nil
	}
	defer file.Close()
	img,_,err := image.Decode(file)
	if err != nil {
		logfile.WriteString(err.Error())
		Errors <- err
		return nil
	}
	return img
}

func resizeBG(img image.Image,CI *ConnInfo) image.Image {
	return resize.Resize(uint(CI.Screen.WidthInPixels),uint(CI.Screen.HeightInPixels),img,resize.Lanczos3)
}

func changeFormatBG(img image.Image,CI *ConnInfo) (*xgraphics.Image) {
	ximg:= xgraphics.NewConvert(CI.XConn,img)
	return ximg
}

// func DrawBackground(CI *ConnInfo,ximg []uint8,y int16,gc xproto.Gcontext,background xproto.Pixmap,logfile *os.File) {
// 	cookie := xproto.PutImageChecked(
// 		CI.Conn,
// 		xproto.ImageFormatZPixmap,
// 		xproto.Drawable(background),
// 		gc,
// 		CI.Screen.WidthInPixels,
// 		8,
// 		0,y,
// 		0,
// 		CI.Screen.RootDepth,
// 		ximg,
// 	).Check()
// 	if cookie != nil {
// 		logfile.WriteString(cookie.Error())
// 		Errors <- cookie
// 	}
// 	wg.Done()
// }

// func DrawAllBG(CI *ConnInfo,ximg xgraphics.Image,gc xproto.Gcontext,background xproto.Pixmap,logfile *os.File) {
// 	packageSize := 7680 * 8
// 	totalSize := len(ximg.Pix)
// 	y := 0
// 	for start := 0;start < totalSize;start += packageSize {
// 		wg.Add(1)
// 		end := start + packageSize
// 		go DrawBackground(CI,ximg.Pix[start:end],int16(y),gc,background,logfile)
// 		y += 8
// 	}
// 	wg.Wait()
// }

// func CreateBG(CI *ConnInfo, path string,logfile *os.File) xproto.Pixmap {
// 	background := CreatePixelMap(CI,logfile)
// 	gc := CreateGCtx(CI,logfile)
// 	defer xproto.FreeGC(CI.Conn,gc)

// 	img:= GetBG(path,logfile)
// 	rimg := resizeBG(img,CI)
// 	ximg := changeFormatBG(rimg,CI)
// 	DrawAllBG(CI,*ximg,gc,background,logfile)

// 	return  background
// }

func CreateBG(CI *ConnInfo, logfile *os.File,path string)  xproto.Pixmap{
	background := CreatePixelMap(CI,logfile)
	img:= GetBG(path,logfile)
	rimg := resizeBG(img,CI)
	ximg := changeFormatBG(rimg,CI)
	ximg.XPaint(CI.Screen.Root)
	return background
}


func GetKeyMap(CI *ConnInfo) *xproto.GetKeyboardMappingReply {
	minCode := CI.Setup.MinKeycode
	maxCode := CI.Setup.MaxKeycode
	count := byte(maxCode - minCode + 1)

	reply,err := xproto.GetKeyboardMapping(CI.Conn,minCode,count).Reply()
	Errors <- err

	return reply
}

func CheckKeyCode(CI *ConnInfo, reply *xproto.GetKeyboardMappingReply,keysum uint32) xproto.Keycode {
	perCode := int(reply.KeysymsPerKeycode)
	minCode := CI.Setup.MinKeycode

	for i:= 0;i < len(reply.Keysyms);i += perCode {
		currentKeyCode := minCode + xproto.Keycode(i/perCode)

		for j:=0; j < perCode;j++ {
			sym := reply.Keysyms[i+j]

			if uint32(sym) == keysum {
				return xproto.Keycode(currentKeyCode)
			}
		}
	}
	return 0
}


func CreateConnect() ConnInfo{
	conn,err := xgb.NewConn()
	if err != nil {
		log.Printf("Ошибка! %v\n",err)
	}

	xbutl,err := xgbutil.NewConnXgb(conn)
	if err != nil {
		log.Printf("Ошибка! %v",err)
	}

	setup := xproto.Setup(conn)
	screen := setup.Roots[0]

	CI := ConnInfo{
		Conn: conn,
		XConn: xbutl,
		Setup: setup,
		Screen: screen,
	}

	return CI
}

func ChangeScreenRoot(CI *ConnInfo,logfile *os.File) xproto.Window {
	background:= CreateBG(CI,logfile,cfg.BackgroundPath)
	
	evMask := uint32(xproto.CwBackPixmap | xproto.CwEventMask)
	root_vallist := []uint32{
			uint32(background),
			xproto.EventMaskExposure | xproto.EventMaskKeyPress | xproto.EventMaskSubstructureRedirect,
		}
	xproto.ChangeWindowAttributes(CI.Conn,CI.Screen.Root,evMask,root_vallist)
	xproto.ClearArea(CI.Conn,false,CI.Screen.Root,0,0,CI.Screen.WidthInPixels,CI.Screen.HeightInPixels)


	return CI.Screen.Root
}

func raiseWindow(CI *ConnInfo,wid xproto.Window)  {
	mask := uint16(xproto.ConfigWindowStackMode)
	valist := []uint32{
		uint32(xproto.StackModeAbove),
	}
	xproto.ConfigureWindow(CI.Conn,wid,mask,valist)
}

func FocusOn(CI *ConnInfo,wid xproto.Window) {
	times := xproto.Timestamp(xproto.TimeCurrentTime)

	xproto.SetInputFocusChecked(CI.Conn,xproto.InputFocusParent,wid,times)
}


func ConfigLoad() *config.Config{
	var err error
	cfg,err = config.LoadConfig("/home/mark/VSCodeProjects/GWM/config.yaml")
	if err != nil {
		Errors <- err
	}

	return cfg
}

func ChangeNewWindowAttr(CI *ConnInfo,wid xproto.Window) {
	mask := uint32(xproto.CwEventMask)
	valist := []uint32{
		uint32(xproto.EventMaskEnterWindow),
	}

	xproto.ChangeWindowAttributes(CI.Conn,wid,mask,valist)
	xevent.EnterNotifyFun(func(xu *xgbutil.XUtil,event xevent.EnterNotifyEvent) {
		raiseWindow(CI,wid)
		FocusOn(CI,wid)
	}).Connect(CI.XConn,wid)	

	xproto.MapWindow(CI.Conn,wid)
}

func spawn() {
	term := exec.Command(cfg.TerminalConfig.Terminal)
	term.Run()
}

func EventChecker(CI *ConnInfo,wid xproto.Window,logfile *os.File) {
	//KeyPress Callback
	xevent.KeyPressFun(func(xu *xgbutil.XUtil, event xevent.KeyPressEvent) {
		if event.Detail == keycode || uint16(event.State)&xproto.ModMaskShift != 0 {
			logfile.WriteString(string(event.Detail))
			spawn()
		}
	}).Connect(CI.XConn,wid)
	
	logfile.WriteString("Обработка нажатий включена!")

	//Window Creating
	xevent.MapRequestFun(func(xu *xgbutil.XUtil,event xevent.MapRequestEvent) {
		ChangeNewWindowAttr(CI,event.Window)
	}).Connect(CI.XConn,wid)

	logfile.WriteString("Обработка мапреков включена!")
}
func Debug() {
	logfile,_ := os.OpenFile("/home/mark/VSCodeProjects/GWM/logs.txt",os.O_WRONLY,0644)
	defer logfile.Close()
	for err := range Errors {
		logfile.WriteString(err.Error())
	}
}

func main() {
	logfile,_ := os.OpenFile("/home/mark/VSCodeProjects/GWM/logs.txt",os.O_WRONLY,0644)
	defer logfile.Close()

	// go Debug()
	logfile.WriteString("Debug On!\n")
	cfg = ConfigLoad()
	logfile.WriteString("Config Load!\n")
	CI := CreateConnect()
	logfile.WriteString("Connection Up!\n")
	wid := ChangeScreenRoot(&CI,logfile)
	logfile.WriteString("Root Screnn Changed!\n")

	reply := GetKeyMap(&CI)
	logfile.WriteString("Get Keymap!\n")
	
	keycode = CheckKeyCode(&CI,reply,cfg.TerminalConfig.TermHotKey)
	logfile.WriteString("Get KeyCode!\n")
	EventChecker(&CI,wid,logfile)
	logfile.Sync()

	logfile.WriteString("EventChecker On!\n")
	xevent.Main(CI.XConn)
}
