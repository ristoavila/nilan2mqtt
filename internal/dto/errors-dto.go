package dto

import "github.com/ristoavila/nilan"

type ErrorsDTO struct {
	OldFilter   string `json:"old_filter"`
	OtherErrors string `json:"other_errors"`
}

const ErrorsTopic string = "nilan/errors"

func CreateErrorsDTO(errors nilan.Errors) ErrorsDTO {
	return ErrorsDTO{
		OldFilter:   onOffString(errors.OldFilterWarning),
		OtherErrors: onOffString(errors.OtherErrors),
	}
}

func onOffString(on bool) string {
	if on {
		return "ON"
	} else {
		return "OFF"
	}
}
