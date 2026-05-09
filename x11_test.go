package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"testing"
	"github.com/jezek/xgb" 
	"github.com/jezek/xgb/xproto"
)


var wid xproto.Window
func Connect() {
	X,_ = xgb.NewConn()
	setup = xproto.Setup(X)
	screen = setup.Roots[0]
	CreatePixelMap(X,&screen)
	CreateGC(X,&screen)
	wid,_ = xproto.NewWindowId(X)
}

func TestGetBG(t *testing.T) {
	type BGtypes struct {
		name string	
		InputPath string
		err error
	}

	var Results []BGtypes
	folder := "backgrounds"
	ent,_ := os.ReadDir(folder)
	for _,entry := range ent {
		numtest := 1
		name := entry.Name()
		Results = append(Results,BGtypes{
			name: fmt.Sprintf("Тест номер %v",numtest),
			InputPath: "./backgrounds/" + name,
			err: nil,
	})
		numtest++
	}

	t.Logf("Длина тестов : %d",len(Results))
	Connect()
	for _,tt := range Results {
		t.Run(tt.name,func(t *testing.T){
			err := CreateBG(X,screen,tt.InputPath,wid)
			if err != tt.err {
				t.Errorf("Ошибка! %v",err)
			}
		})
	}
	
}


