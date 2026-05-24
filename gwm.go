package main

import (
	"fmt"
	"gwm/config"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/xevent"
	"github.com/jezek/xgbutil/xgraphics"
	"github.com/nfnt/resize"
)

var wg sync.WaitGroup 
var keycode xproto.Keycode
var cfg *config.Config

type ConnInfo struct {
	Conn *xgb.Conn
	XConn *xgbutil.XUtil
	Setup *xproto.SetupInfo
	Screen xproto.ScreenInfo
}


func CreatePixelMap(CI *ConnInfo) (xproto.Pixmap,error) {
	background,err := xproto.NewPixmapId(CI.Conn)
	if err != nil {
		return background,fmt.Errorf("Ошибка!: %v",err)
	}
	xproto.CreatePixmap(CI.Conn, CI.Screen.RootDepth, background, xproto.Drawable(CI.Screen.Root),CI.Screen.WidthInPixels,CI.Screen.HeightInPixels)
	return background,nil
}

func CreateGCtx(CI *ConnInfo) (xproto.Gcontext,error) {
	gc,err := xproto.NewGcontextId(CI.Conn)
	if err != nil {
		return gc,fmt.Errorf("Ошибка GC!: %v",err)
	}
	xproto.CreateGC(CI.Conn,gc,xproto.Drawable(CI.Screen.Root),0,[]uint32{})
	return gc,nil
}

func GetBG(path string) (image.Image,error) {
	file,err := os.Open(path)
	if err != nil {
		return nil,fmt.Errorf("Ошибка открытия изображения!: %v",err)
	}
	defer file.Close()
	img,_,err := image.Decode(file)
	if err != nil {
		return nil,fmt.Errorf("Ошибка при декодировании изображения!: %v",err)
	}
	return img,nil
}

func resizeBG(img image.Image,CI *ConnInfo) image.Image {
	return resize.Resize(uint(CI.Screen.WidthInPixels),uint(CI.Screen.HeightInPixels),img,resize.Lanczos3)
}

func changeFormatBG(img image.Image,CI *ConnInfo) (*xgraphics.Image) {
	ximg:= xgraphics.NewConvert(CI.XConn,img)
	return ximg
}

func DrawBackground(CI *ConnInfo,ximg []uint8,y int16,gc xproto.Gcontext,background xproto.Pixmap) {
	xproto.PutImage(
		CI.Conn,
		xproto.ImageFormatZPixmap,
		xproto.Drawable(background),
		gc,
		CI.Screen.WidthInPixels,
		8,
		0,y,
		0,
		CI.Screen.RootDepth,
		ximg,
	)
	wg.Done()
}

func DrawAllBG(CI *ConnInfo,ximg xgraphics.Image,gc xproto.Gcontext,background xproto.Pixmap) {
	packageSize := 7680 * 8
	totalSize := len(ximg.Pix)
	y := 0
	for start := 0;start < totalSize;start += packageSize {
		wg.Add(1)
		end := start + packageSize
		go DrawBackground(CI,ximg.Pix[start:end],int16(y),gc,background)
		y += 8
	}
	wg.Wait()
}

func CreateBG(CI *ConnInfo, path string) (xproto.Pixmap,error) {
	background,err := CreatePixelMap(CI)
	if err != nil {
		return background,fmt.Errorf("Ошибка при создании pixmap!: %v",err)
	}

	gc,err2 := CreateGCtx(CI)
	if err2 != nil {
		log.Printf("Ошибка!: %v",err2)
	}
	defer xproto.FreeGC(CI.Conn,gc)

	img,err := GetBG(path)
	if err != nil {
		return background,fmt.Errorf("Ошибка при чтении файла заднего фона!: %v",err)
	}

	rimg := resizeBG(img,CI)

	ximg := changeFormatBG(rimg,CI)

	DrawAllBG(CI,*ximg,gc,background)

	return  background,nil
}

func GetKeyMap(CI *ConnInfo) (*xproto.GetKeyboardMappingReply,error) {
	minCode := CI.Setup.MinKeycode
	maxCode := CI.Setup.MaxKeycode
	count := byte(maxCode - minCode + 1)

	reply,err := xproto.GetKeyboardMapping(CI.Conn,minCode,count).Reply()
	if err != nil {
		return nil,fmt.Errorf("Ошибка %v",err)
	}

	return reply,nil
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

func ChangeScreenRoot(CI *ConnInfo) (xproto.Window,error){
	background,err := CreateBG(CI,cfg.BackgroundPath)
 	if err != nil {
 		log.Printf("Ошибка! %v",err)
 	}

	evMask := uint32(xproto.CwBackPixmap | xproto.CwEventMask)
	root_vallist := []uint32{
			uint32(background),
			xproto.EventMaskExposure | xproto.EventMaskKeyPress | xproto.EventMaskSubstructureRedirect,
		}
	xproto.ChangeWindowAttributes(CI.Conn,CI.Screen.Root,evMask,root_vallist)
	xproto.ClearArea(CI.Conn,false,CI.Screen.Root,0,0,CI.Screen.WidthInPixels,CI.Screen.HeightInPixels)


	
	return CI.Screen.Root,nil
}

func raiseWindow(CI *ConnInfo,wid xproto.Window)  {
	mask := uint16(xproto.ConfigWindowStackMode)
	valist := []uint32{
		uint32(xproto.StackModeAbove),
	}
	fmt.Print("Окно!\n")
	xproto.ConfigureWindowChecked(CI.Conn,wid,mask,valist).Check()
}

func FocusOn(CI *ConnInfo,wid xproto.Window) {
	times := xproto.Timestamp(xproto.TimeCurrentTime)

	xproto.SetInputFocusChecked(CI.Conn,xproto.InputFocusParent,wid,times)
}


func ConfigLoad() *config.Config{
	cfg,err := config.LoadConfig("/home/mark/VSCodeProjects/GWM/config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига! %v",err)
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

	term.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	term.ExtraFiles = nil
	term.Env = os.Environ()
	term.Start()

	go func() {
		term.Wait()
	}()
}

func EventChecker(CI *ConnInfo,wid xproto.Window) {
	//KeyPress Callback
	xevent.KeyPressFun(func(xu *xgbutil.XUtil, event xevent.KeyPressEvent) {
		if event.Detail == keycode || uint16(event.State)&xproto.ModMaskShift != 0 {
			spawn()
		}
	}).Connect(CI.XConn,wid)

	//Window Creating
	xevent.MapRequestFun(func(xu *xgbutil.XUtil,event xevent.MapRequestEvent) {
		ChangeNewWindowAttr(CI,event.Window)
	}).Connect(CI.XConn,wid)
}

func main() {
	cfg = ConfigLoad()
	CI := CreateConnect()
	wid,err := ChangeScreenRoot(&CI)
	if err != nil {
		log.Fatal(err)
	}

	reply,err := GetKeyMap(&CI)
	if err != nil {
		log.Fatal("Ошибка!",err)
	}

	keycode = CheckKeyCode(&CI,reply,cfg.TerminalConfig.TermHotKey)
	EventChecker(&CI,wid)
	xevent.Main(CI.XConn)
}
