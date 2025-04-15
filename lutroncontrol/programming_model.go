package main

import (
	"context"

	"github.com/unixpickle/essentials"
)

type rawProgrammingModel struct {
	Href                 string `json:"href"`
	ProgrammingModelType string
	Direction            *string
	Preset               *rawLink
	DualActionProperties *struct {
		PressPreset   rawLink
		ReleasePreset rawLink
	}
}

type rawPresetInner struct {
	Href                     string `json:"href"`
	DimmedLevelAssignment    *rawLink
	DimmedLevelAssignments   []rawLink
	SwitchedLevelAssignment  *rawLink
	SwitchedLevelAssignments []rawLink
}

func (r *rawPresetInner) AllDimmedLevelAssignments() []rawLink {
	if r.DimmedLevelAssignment != nil {
		return append(r.DimmedLevelAssignments, *r.DimmedLevelAssignment)
	}
	return r.DimmedLevelAssignments
}

func (r *rawPresetInner) AllSwitchedLevelAssignments() []rawLink {
	if r.SwitchedLevelAssignment != nil {
		return append(r.SwitchedLevelAssignments, *r.SwitchedLevelAssignment)
	}
	return r.SwitchedLevelAssignments
}

type rawPreset struct {
	Preset rawPresetInner
}

type rawDimmedLevelAssignment struct {
	DimmedLevelAssignment struct {
		Href      string `json:"href"`
		FadeTime  string
		DelayTime string
		Level     int
	}
}

type rawSwitchedLevelAssignment struct {
	SwitchedLevelAssignment struct {
		Href          string `json:"href"`
		DelayTime     string
		SwitchedLevel string
	}
}

type ProgrammingModel struct {
	Href                 string
	ProgrammingModelType string
	Direction            *string `json:",omitempty"`
	Preset               *Preset `json:",omitempty"`
	PressPreset          *Preset `json:",omitempty"`
	ReleasePreset        *Preset `json:",omitempty"`
}

type DimmedLevelAssignment struct {
	Href      string
	FadeTime  string
	DelayTime string
	Level     int
}

type SwitchedLevelAssignment struct {
	Href          string
	DelayTime     string
	SwitchedLevel string
}

type Preset struct {
	Href                     string
	DimmedLevelAssignments   []DimmedLevelAssignment
	SwitchedLevelAssignments []SwitchedLevelAssignment
}

func GetProgrammingModels(ctx context.Context, conn BrokerConn) (models map[string]*ProgrammingModel, err error) {
	defer essentials.AddCtxTo("get programming models", &err)

	var modelsResponse struct {
		ProgrammingModels []rawProgrammingModel
	}
	if err := ReadRequest(ctx, conn, "/programmingmodel", &modelsResponse); err != nil {
		return nil, err
	}
	allPresetURLs := map[string]struct{}{}
	for _, model := range modelsResponse.ProgrammingModels {
		if model.Preset != nil {
			allPresetURLs[model.Preset.Href] = struct{}{}
		} else if model.DualActionProperties != nil {
			allPresetURLs[model.DualActionProperties.PressPreset.Href] = struct{}{}
			allPresetURLs[model.DualActionProperties.ReleasePreset.Href] = struct{}{}
		}
	}
	presets, err := ReadRequestsAsMap[rawPreset](ctx, conn, allPresetURLs)
	if err != nil {
		return nil, err
	}

	allDimmedLevelAssignmentURLs := map[string]struct{}{}
	allSwitchedLevelAssignmentURLs := map[string]struct{}{}
	for _, preset := range presets {
		for _, x := range preset.Preset.AllDimmedLevelAssignments() {
			allDimmedLevelAssignmentURLs[x.Href] = struct{}{}
		}
		for _, x := range preset.Preset.AllSwitchedLevelAssignments() {
			allSwitchedLevelAssignmentURLs[x.Href] = struct{}{}
		}
	}
	dimmedLevelAssignments, err := ReadRequestsAsMap[rawDimmedLevelAssignment](ctx, conn, allDimmedLevelAssignmentURLs)
	if err != nil {
		return nil, err
	}
	switchedLevelAssignments, err := ReadRequestsAsMap[rawSwitchedLevelAssignment](ctx, conn, allSwitchedLevelAssignmentURLs)
	if err != nil {
		return nil, err
	}

	getPresetForURL := func(url string) *Preset {
		rawPreset, ok := presets[url]
		if !ok {
			return nil
		}
		dimmed := []DimmedLevelAssignment{}
		switched := []SwitchedLevelAssignment{}
		for _, d := range rawPreset.Preset.AllDimmedLevelAssignments() {
			if d1, ok := dimmedLevelAssignments[d.Href]; ok {
				dimmed = append(dimmed, DimmedLevelAssignment(d1.DimmedLevelAssignment))
			}
		}
		for _, s := range rawPreset.Preset.AllSwitchedLevelAssignments() {
			if s1, ok := switchedLevelAssignments[s.Href]; ok {
				switched = append(switched, SwitchedLevelAssignment(s1.SwitchedLevelAssignment))
			}
		}
		return &Preset{
			Href:                     rawPreset.Preset.Href,
			DimmedLevelAssignments:   dimmed,
			SwitchedLevelAssignments: switched,
		}
	}

	results := map[string]*ProgrammingModel{}
	for _, model := range modelsResponse.ProgrammingModels {
		outModel := &ProgrammingModel{
			Href:                 model.Href,
			ProgrammingModelType: model.ProgrammingModelType,
			Direction:            model.Direction,
		}
		if model.Preset != nil {
			outModel.Preset = getPresetForURL(model.Preset.Href)
		} else if model.DualActionProperties != nil {
			outModel.PressPreset = getPresetForURL(model.DualActionProperties.PressPreset.Href)
			outModel.ReleasePreset = getPresetForURL(model.DualActionProperties.ReleasePreset.Href)
		}
		results[model.Href] = outModel
	}

	return results, nil
}
