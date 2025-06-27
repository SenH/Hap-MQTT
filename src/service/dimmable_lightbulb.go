package service

import (
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

type DimmableLightbulb struct {
	*service.S
	On         *characteristic.On
	Brightness *characteristic.Brightness
	Reachable  *characteristic.Reachable
}

func NewDimmableLightbulb() *DimmableLightbulb {
	s := DimmableLightbulb{}
	s.S = service.New(service.TypeLightbulb)

	s.On = characteristic.NewOn()
	s.AddC(s.On.C)

	s.Brightness = characteristic.NewBrightness()
	s.AddC(s.Brightness.C)

	return &s
}
