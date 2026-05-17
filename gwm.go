package main
import (
	"github.com/jezek/xgb" 
	"github.com/jezek/xgb/xproto" 
	"log"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"github.com/nfnt/resize"
	"github.com/jezek/xgbutil"
	"github.com/jezek/xgbutil/xgraphics"
	"sync"
)

var wg sync.WaitGroup

type ConnInfo struct {
	Conn *xgb.Conn
	XConn *xgbutil.XUtil
	Setup *xproto.SetupInfo
	Screen xproto.ScreenInfo
}

const (
	XK_Shift_L = 0xffe1  /* Left shift */
   	XK_Shift_R = 0xffe2  /* Right shift */
   	XK_Control_L = 0xffe3  /* Left control */
   	XK_Control_R = 0xffe4  /* Right control */
   	XK_Caps_Lock = 0xffe5  /* Caps lock */
   	XK_Shift_Lock = 0xffe6  /* Shift lock */
)

func CreatePixelMap(CI *ConnInfo) (xproto.Pixmap,error) {
	background,err := xproto.NewPixmapId(CI.Conn)
	if err != nil {
		return background,fmt.Errorf("Ошибка!: %v",err)
	}
	xproto.CreatePixmap(CI.Conn, CI.Screen.RootDepth, background, xproto.Drawable(CI.Screen.Root),CI.Screen.WidthInPixels,CI.Screen.HeightInPixels)
	return background,nil
}

func CreateGC(CI *ConnInfo) (xproto.Gcontext,error) {
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

	gc,err2 := CreateGC(CI)
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


func Connect() ConnInfo{
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

func CreateWindow(CI *ConnInfo) (error){
	wid,err := xproto.NewWindowId(CI.Conn)
	if err != nil {
		return fmt.Errorf("Проблема с id!: %v",err)
	}

	background,err := CreateBG(CI,"/home/mark/VSCodeProjects/GWM/backgrounds/bg1.jpg")
	if err != nil {
		log.Printf("Ошибка! %v",err)
	}
	xproto.CreateWindow(
		CI.Conn,
		CI.Screen.RootDepth,
		wid,
		CI.Screen.Root,
		0,0,
		CI.Screen.WidthInPixels,CI.Screen.HeightInPixels,
		0,
		xproto.WindowClassInputOutput,
		CI.Screen.RootVisual,

		xproto.CwBackPixmap | xproto.CwEventMask,
		[]uint32{
			uint32(background),
			xproto.EventMaskExposure | xproto.EventMaskKeyPress,
		},
	)
	
	defer xproto.FreePixmap(CI.Conn,background) 
	xproto.MapWindow(CI.Conn,wid)
	return nil
}

func main() {
	CI := Connect()
	err := CreateWindow(&CI)
	if err != nil {
		log.Fatal(err)
	}

	reply,err := GetKeyMap(&CI)
	if err != nil {
		log.Fatal("Ошибка!",err)
	}

	keycode := CheckKeyCode(&CI,reply,XK_Caps_Lock)
	fmt.Print(keycode)

	for {
		_,err := CI.Conn.WaitForEvent()
		if err != nil {
			log.Fatal("Ошибка!",err)
			return
		}	
	}
}
