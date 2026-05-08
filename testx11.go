package main
import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"log"
	"fmt"
)

func start() (*xgb.Conn,error) {
	X,err := xgb.NewConn()
	if err != nil {
		return nil,fmt.Errorf("Не удалось подключиться!: %v",err)
	}
	setup := xproto.Setup(X)
	screen := setup.Roots[0]

	wid,err := xproto.NewWindowId(X)
	if err != nil {
		return nil,fmt.Errorf("Проблема с id!: %v",err)
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
	return X,nil
}

func main() {
	conn,err := start()
	if err != nil {
		log.Fatal(err)
	}
	
	for {
		ev,err := conn.WaitForEvent()
		if err != nil {
			log.Fatal("Ошибка!",err)
			return
		}
		fmt.Println(ev)
	}
}