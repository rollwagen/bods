package main

import (
	"encoding/base64"
	"fmt"
	"image"
)

// max width of an image is 8000 pixels; see https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-anthropic-claude-messages.html
const maxSizeImage = 8000

func validateImageDimentions(img image.Image, format string) (bool, string) {
	x := img.Bounds().Max.X
	y := img.Bounds().Max.Y
	logger.Printf("%d x %d size of image\n", x, y)
	if img.Bounds().Max.X > maxSizeImage || img.Bounds().Max.Y > maxSizeImage {
		msg := fmt.Sprintf("the maximum height and width of an image is %d pixels. %s has size %d x %d", maxSizeImage, format, x, y)
		logger.Println(msg)
		return false, msg
	}
	return true, ""
}

func imgToMessageContent(imgBytes []byte, imgType string) Content {
	b64Img := base64.StdEncoding.EncodeToString(imgBytes)
	s := Source{
		Type:      "base64",
		MediaType: imgType,
		Data:      b64Img,
	}
	imageContent := Content{
		Type:   MessageContentTypeImage,
		Source: &s,
	}

	return imageContent
}
