package main

import (
	"context"
	"fmt"

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
}

type rawZoneStatus struct {
	Href           string `json:"href"`
	Level          int
	Zone           rawLink
	StatusAccuracy string
}

type DeviceInfo struct {
	FullyQualifiedName []string
	DeviceType         string
	Level              *int
	Zone               *string
}

func GetDevices(ctx context.Context, conn BrokerConn) (devices []*DeviceInfo, err error) {
	defer essentials.AddCtxTo("get devices", &err)

	var devicesResponse struct {
		Devices []rawDevice
	}
	if err := ReadRequest(ctx, conn, "/device", &devicesResponse); err != nil {
		return nil, err
	}
	fmt.Println("devices", devicesResponse)

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
		devices = append(devices, outDev)
	}

	return devices, nil
}
