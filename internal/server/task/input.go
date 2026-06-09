package task

import (
	"fmt"

	"github.com/yinxulai/ait/internal/server/prompt"
	"github.com/yinxulai/ait/internal/server/types"
)

func HydrateInput(input types.Input) (types.Input, error) {
	if input.PromptSource != nil {
		return input, nil
	}

	switch input.PromptMode {
	case "", "text":
		if input.PromptText == "" {
			return input, fmt.Errorf("prompt_text is required for prompt_mode=text")
		}
		source, err := prompt.LoadPrompts(input.PromptText)
		if err != nil {
			return input, err
		}
		input.PromptSource = source
	case "file":
		if input.PromptFile == "" {
			return input, fmt.Errorf("prompt_file is required for prompt_mode=file")
		}
		source, err := prompt.LoadPromptsFromFile(input.PromptFile)
		if err != nil {
			return input, err
		}
		input.PromptSource = source
	case "generated":
		if input.PromptLength <= 0 {
			return input, fmt.Errorf("prompt_length must be greater than zero for prompt_mode=generated")
		}
		source, err := prompt.LoadPromptByLength(input.PromptLength)
		if err != nil {
			return input, err
		}
		input.PromptSource = source
	case "raw":
		if input.PromptText == "" {
			return input, fmt.Errorf("prompt_text is required for prompt_mode=raw (paste the raw JSON request body)")
		}
		source, err := prompt.LoadPrompts(input.PromptText)
		if err != nil {
			return input, err
		}
		input.PromptSource = source
	default:
		return input, fmt.Errorf("unsupported prompt_mode: %s", input.PromptMode)
	}

	return input, nil
}
