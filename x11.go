//Импорт и тп
package main
import (
	"github.com/jezek/xgb" //Главная библиотека для работы с x11
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
)

//Объявление переменых
var X *xgb.Conn //Подключение
var setup *xproto.SetupInfo //Ифнормация о сессии
var screen xproto.ScreenInfo //Информация о экране
var background xproto.Pixmap //Задний фон
var xbutil *xgbutil.XUtil
var gc xproto.Gcontext

func CreatePixelMap(X *xgb.Conn,screen *xproto.ScreenInfo) (*xproto.Pixmap,error) {
	var err error
	background,err = xproto.NewPixmapId(X)
	if err != nil {
		return nil,fmt.Errorf("Ошибка!: %v",err)
	}
	xproto.CreatePixmap(X, screen.RootDepth, background, xproto.Drawable(screen.Root),screen.WidthInPixels,screen.HeightInPixels)
	return &background,nil
}

func CreateGC(X *xgb.Conn,screen *xproto.ScreenInfo) error {
	var err error
	gc,err = xproto.NewGcontextId(X)
	if err != nil {
		return fmt.Errorf("Ошибка GC!: %v",err)
	}
	xproto.CreateGC(X,gc,xproto.Drawable(screen.Root),0,[]uint32{})
	return nil
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

func resizeBG(img image.Image,screen xproto.ScreenInfo,wid xproto.Window) image.Image {
	return resize.Resize(uint(screen.WidthInPixels),uint(screen.HeightInPixels),img,resize.NearestNeighbor)
}

func changeFormatBG(img image.Image,X *xgb.Conn) (*xgraphics.Image,error) {
	ximg:= xgraphics.NewConvert(xbutil,img)
	return ximg,nil

}

func DrawBackground(ximg []uint8,y int16) {
	xproto.PutImage(
		X,
		xproto.ImageFormatZPixmap,
		xproto.Drawable(background),
		gc,
		screen.WidthInPixels,
		8,
		0,y,
		0,
		screen.RootDepth,
		ximg,
	)
}

func DrawAllBG(ximg xgraphics.Image) {
	packageSize := 7680 * 8
	totalSize := len(ximg.Pix)
	y := 0
	for start := 0;start < totalSize;start += packageSize {
		end := start + packageSize
		DrawBackground(ximg.Pix[start:end],int16(y))
		y += 8
	}
}

func CreateBG(X *xgb.Conn, screen xproto.ScreenInfo, path string,wid xproto.Window) error {
	_,err := CreatePixelMap(X,&screen)
	if err != nil {
		return fmt.Errorf("Ошибка при создании pixmap!: %v",err)
	}

	err2 := CreateGC(X,&screen)
	if err2 != nil {
		log.Printf("Ошибка!: %v",err2)
	}

	img,err := GetBG(path)
	if err != nil {
		return fmt.Errorf("Ошибка при чтении файла заднего фона!: %v",err)
	}

	rimg := resizeBG(img,screen,wid)

	ximg,err := changeFormatBG(rimg,X)
	if err != nil {
		return fmt.Errorf("Ошибка при изменении формата заднего фона!: %v",err)
	}

	DrawAllBG(*ximg)

	return  nil
}


func start() (xproto.Window,error) {
	var err error
	X,err = xgb.NewConn()
	xbutil,err = xgbutil.NewConnXgb(X)
	if err != nil {
		return 0,fmt.Errorf("Не удалось подключиться!: %v",err)
	}
	setup = xproto.Setup(X)
	screen = setup.Roots[0]

	wid,err := xproto.NewWindowId(X)
	if err != nil {
		return 0,fmt.Errorf("Проблема с id!: %v",err)
	}

	CreateBG(X,screen,"./backgrounds/background.jpg",wid)
	xproto.CreateWindow(
		X,
		screen.RootDepth,
		wid,
		screen.Root,
		0,0,
		screen.WidthInPixels,screen.HeightInPixels,
		0,
		xproto.WindowClassInputOutput,
		screen.RootVisual,

		xproto.CwBackPixmap | xproto.CwEventMask,
		[]uint32{
			uint32(background),
			xproto.EventMaskExposure | xproto.EventMaskKeyPress,
		},
	)
	
	xproto.MapWindow(X,wid)
	return wid,nil
}

func main() {
	_,err := start()
	if err != nil {
		log.Fatal(err)
	}

	
	
	for {
		_,err := X.WaitForEvent()
		if err != nil {
			log.Fatal("Ошибка!",err)
			return
		}	
	}
}