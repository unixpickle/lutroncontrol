package main

import (
	"context"

	"github.com/unixpickle/essentials"
)

type rawLink struct {
	Href string `json:"href"`
}

type rawDevice struct {
	FullyQualifiedName []string
	LocalZones         []rawLink
	AssociatedArea     *rawLink
	DeviceType         string
	ButtonGroups       []rawLink
}

type rawZoneStatus struct {
	Href           string `json:"href"`
	Level          int
	Zone           rawLink
	StatusAccuracy string
}

type rawButton struct {
	Href         string `json:"href"`
	Name         string
	ButtonNumber int
	Parent       rawLink
}

type ButtonInfo struct {
	Href         string
	Name         string
	ButtonNumber int
}

type DeviceInfo struct {
	FullyQualifiedName []string
	DeviceType         string
	Level              *int `json:",omitempty"`
	Zone               *string
	Buttons            []*ButtonInfo `json:",omitempty"`
}

func GetDevices(ctx context.Context, conn BrokerConn) (devices []*DeviceInfo, err error) {
	defer essentials.AddCtxTo("get devices", &err)

	var devicesResponse struct {
		Devices []rawDevice
	}
	if err := ReadRequest(ctx, conn, "/device", &devicesResponse); err != nil {
		return nil, err
	}

	var zoneResponse struct {
		ZoneStatuses []rawZoneStatus
	}
	if err := ReadRequest(ctx, conn, "/zone/status", &zoneResponse); err != nil {
		return nil, err
	}
	zoneToLevel := map[string]rawZoneStatus{}
	for _, zone := range zoneResponse.ZoneStatuses {
		zoneToLevel[zone.Zone.Href] = zone
	}

	var buttonResponse struct {
		Buttons []rawButton
	}
	if err := ReadRequest(ctx, conn, "/button", &buttonResponse); err != nil {
		return nil, err
	}
	buttonGroupToButtons := map[string][]*ButtonInfo{}
	for _, button := range buttonResponse.Buttons {
		key := button.Parent.Href
		buttonInfo := &ButtonInfo{
			Href:         button.Href,
			Name:         button.Name,
			ButtonNumber: button.ButtonNumber,
		}
		buttonGroupToButtons[key] = append(buttonGroupToButtons[key], buttonInfo)
	}

	for _, device := range devicesResponse.Devices {
		outDev := &DeviceInfo{
			FullyQualifiedName: device.FullyQualifiedName,
			DeviceType:         device.DeviceType,
		}
		for _, zone := range device.LocalZones {
			if info, ok := zoneToLevel[zone.Href]; ok {
				outDev.Level = &info.Level
				outDev.Zone = &zone.Href
			}
		}
		for _, buttonGroup := range device.ButtonGroups {
			outDev.Buttons = append(outDev.Buttons, buttonGroupToButtons[buttonGroup.Href]...)
		}
		devices = append(devices, outDev)
	}

	return devices, nil
}
