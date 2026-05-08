//Импорт и тп
package main
import (
	"github.com/jezek/xgb" //Главная библиотека для работы с x11
	"github.com/jezek/xgb/xproto" 
	"log"
	"fmt"
)

//Объявление переменых
var X *xgb.Conn //Подключение
var setup *xproto.SetupInfo //Ифнормация о сессии
var screen xproto.ScreenInfo //Информация о экране
var background xproto.Pixmap //Задний фон
var gc xproto.Gcontext //Графический контекст

func CreatePixelMap(X *xgb.Conn,screen *xproto.ScreenInfo) error {
	var err error
	background,err = xproto.NewPixmapId(X)
	if err != nil {
		return fmt.Errorf("Ошибка!: %v",err)
	}
	xproto.CreatePixmap(X, screen.RootDepth, background, xproto.Drawable(screen.Root),screen.WidthInPixels,screen.HeightInPixels)
	return nil
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

func start() error {
	var err error
	X,err = xgb.NewConn()
	if err != nil {
		return fmt.Errorf("Не удалось подключиться!: %v",err)
	}
	setup = xproto.Setup(X)
	screen = setup.Roots[0]

	wid,err := xproto.NewWindowId(X)
	if err != nil {
		return fmt.Errorf("Проблема с id!: %v",err)
	}


	err2 := CreatePixelMap(X,&screen)
	if err2 != nil {
		return fmt.Errorf("Ошибка!: %v",err2)
	}
	
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

		xproto.CwBackPixel | xproto.CwEventMask,
		[]uint32{
			screen.WhitePixel,
			xproto.EventMaskExposure | xproto.EventMaskKeyPress,
		},
	)
	
	xproto.MapWindow(X,wid)
	return nil
}

func main() {
	err := start()
	if err != nil {
		log.Fatal(err)
	}

	
	
	for {
		ev,err := X.WaitForEvent()
		if err != nil {
			log.Fatal("Ошибка!",err)
			return
		}
		fmt.Println(ev)
	}
}