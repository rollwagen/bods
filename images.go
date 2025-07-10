package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"net/http"
	"os"
	"slices"
	"strings"
)

// max width of an image is 8000 pixels; see https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-anthropic-claude-messages.html
const maxSizeImage = 8000

func validateImageDimension(img image.Image, format string) (bool, string) {
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

// e.g. input =  file://image1.png,file://image3.jpg
func parseImageURLList(input string) ([]Content, error) {
	var promptContents []Content
	imageURLs := strings.Split(input, ",")
	for _, imageURL := range imageURLs {
		if strings.HasPrefix(imageURL, "file://") {
			logger.Printf("processing image %s\n", imageURL)
			filename := imageURL[7:]
			filename = strings.Trim(filename, `"`)
			filename = strings.Trim(filename, `'`)
			imgBytes, err := os.ReadFile(filename)
			if err != nil {
				return nil, err
			}

			_, imgType, err := validateAndDecodeImage(imgBytes)
			if err != nil {
				return nil, err
			}

			// imgType := http.DetectContentType(imgBytes)
			// if !slices.Contains(MessageContentTypes, imgType) {
			// 	panic("unsupported image type " + imgType)
			// }
			//
			// img, format, err := image.Decode(bytes.NewReader(imgBytes))
			// if err != nil {
			// 	panic("could not decode image " + imgType)
			// }
			//
			// isValidSize, msg := validateImageDimension(img, format)
			// if !isValidSize {
			// 	e := fmt.Errorf("%s", msg)
			// 	return nil, e
			// }

			content := imgToMessageContent(imgBytes, imgType)
			logger.Printf("image conent=%v\n", content)
			promptContents = append(promptContents, content)

		}
	}
	return promptContents, nil
}

func validateAndDecodeImage(imageBytes []byte) (image.Image, string, error) {
	imgType := http.DetectContentType(imageBytes)
	if !slices.Contains(MessageContentTypes, imgType) {
		return nil, "", fmt.Errorf("unsupported image type: %s. Supported types are: %v", imgType, MessageContentTypes)
	}

	img, format, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image of type %s: %w", imgType, err)
	}

	isValidSize, msg := validateImageDimension(img, format)
	if !isValidSize {
		return nil, "", fmt.Errorf("image validation failed: %s", msg)
	}

	return img, imgType, nil
}
