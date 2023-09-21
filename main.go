package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
)

const size = 512

// https://stackoverflow.com/a/62898106/14892663
func isEmoji(aChar rune) bool {
	rangeMin := 127744
	rangeMax := 129782
	rangeMin2 := 126980
	rangeMax2 := 127569
	rangeMin3 := 169
	rangeMax3 := 174
	rangeMin4 := 8205
	rangeMax4 := 12953
	charCode := int(aChar)
	if rangeMin <= charCode && charCode <= rangeMax {
		return true
	} else if rangeMin2 <= charCode && charCode <= rangeMax2 {
		return true
	} else if rangeMin3 <= charCode && charCode <= rangeMax3 {
		return true
	} else if rangeMin4 <= charCode && charCode <= rangeMax4 {
		return true
	}

	return false
}

// Convert a char to its unicode representation
func getCharUnicode(char rune) string {
	return strings.Replace(strings.ToLower(fmt.Sprintf("%U", char)), "u+", "u", -1)
}

func isModifierPair(char1 rune, char2 rune) bool {
	charUnicode1 := strings.ReplaceAll(getCharUnicode(char1), "u", "")

	modifiersFound := validModifiers[charUnicode1]

	if len(modifiersFound) == 0 {
		return false
	}

	charUnicode2 := strings.ReplaceAll(getCharUnicode(char2), "u", "")

	for _, modifier := range modifiersFound {
		if modifier == charUnicode2 {
			return true
		}
	}
	return false
}

func parseEmoji(text string, startI int) (resultSequence []rune, endIndex int) {

	var emojiSequence []rune

	stringRunes := []rune(text)
	joinedLastChar := false

	for i := startI; i < len(stringRunes); i++ {
		char := stringRunes[i]

		if char == joinChar {
			if len(emojiSequence) == 0 {
				fmt.Println("Invalid emoji sequence (start with join char)")
				break
			}
			emojiSequence = append(emojiSequence, char)
			continue
		}
		if isEmoji(char) {
			if len(emojiSequence) == 0 {
				emojiSequence = append(emojiSequence, char)
				continue
			}
			if joinedLastChar {
				break
			}

			// If the last char is a skin modifier and the current char is not a join char, break
			if i != 0 && isSkinToneModifier(stringRunes[i-1]) {
				break
			}

			// If the last char was a join char
			if i != 0 && stringRunes[i-1] == joinChar {
				emojiSequence = append(emojiSequence, char)
				continue
			}

			// If the char is a skin tone modifier
			if isSkinToneModifier(char) {
				emojiSequence = append(emojiSequence, char)
				continue
			}
			// Check if the last char and the current char are a valid modifier pair
			if i != 0 && isModifierPair(stringRunes[i-1], char) {
				emojiSequence = append(emojiSequence, char)
				joinedLastChar = true
				continue
			}
		}
		break
	}

	if len(emojiSequence) == 0 {
		return nil, startI
	}

	// Todo check if the emoji exists
	// If it doesn't, split the sequence by join chars

	return emojiSequence, startI + len(emojiSequence) - 1
}

func buildEmojiFilename(sequence []rune) string {
	var filename string
	for i, char := range sequence {
		charUnicode := getCharUnicode(char)
		if i != 0 {
			charUnicode = strings.ReplaceAll(charUnicode, "u", "")
		}
		filename += charUnicode
		if i != len(sequence)-1 {
			filename += "_"
		}
	}
	return "emoji_" + filename + ".png"
}

//go:embed data/sequences.json
var sequencesJsonData []byte

var validModifiers map[string][]string

//go:embed data/emojis/*
var emojisF embed.FS

//go:embed data/SecularOne.ttf
var secularFont []byte

const replacementChar = rune(65525)
const joinChar = rune(8205) // U+200D

func isSkinToneModifier(char rune) bool {
	// U+1F3FB U+1F3FC U+1F3FD U+1F3FE U+1F3FF
	return char == 127995 || char == 127996 || char == 127997 || char == 127998 || char == 127999
}

func main() {
	// Load all the valid [emoji - modifier] pairs (e.g. "u1f1e6 u1f1e8" -> "flag for United States"))
	if err := json.Unmarshal(sequencesJsonData, &validModifiers); err != nil {
		panic(err)
	}

	startTime := time.Now()

	dc := gg.NewContext(size, size)

	// Load the font face
	f, _ := truetype.Parse(secularFont)
	face := truetype.NewFace(f, &truetype.Options{Size: 50})

	dc.SetFontFace(face)

	dc.SetRGB(0, 0, 0)
	dc.Clear()

	dc.SetRGB(1, 1, 1)

	var emojisUsed []string

	actualString := "hello🍕🍔😀😁😍🙂😶"
	withoutEmojis := ""

	stringRunes := []rune(actualString)
	for i := 0; i < len(stringRunes); i++ {
		char := stringRunes[i]

		if isEmoji(char) {
			result, newI := parseEmoji(actualString, i)
			if len(result) == 0 {
				fmt.Println("No emoji found: " + string(char))
				continue
			}
			filename := buildEmojiFilename(result)
			i = newI

			withoutEmojis += string(replacementChar)
			emojisUsed = append(emojisUsed, filename)

		} else {
			withoutEmojis += string(char)
		}
	}

	fmt.Println(withoutEmojis)

	w, _ := dc.MeasureString(strings.ReplaceAll(withoutEmojis, string(replacementChar), "M"))

	MWidth, _ := dc.MeasureString("M")

	startX := (size - w) / 2
	var x float64 = startX
	var y float64 = size / 2

	emojiIndex := 0
	runes := []rune(withoutEmojis)
	for i := 0; i < len(runes); i++ {

		char := runes[i]

		if char == replacementChar {
			x += MWidth
			if emojiIndex >= len(emojisUsed) {
				fmt.Println("Emoji index out of range")
				continue
			}

			// Get the emoji
			emojiFilename := emojisUsed[emojiIndex]

			// Load the emoji image
			emojiFile, err := emojisF.Open("data/emojis/" + emojiFilename)
			if err != nil {
				fmt.Println("Error loading emoji: " + emojiFilename)
				continue
			}

			emojiImage, _, err1 := image.Decode(emojiFile)

			if err1 != nil {
				continue
			}

			// Resize image
			dstImageFit := imaging.Fit(emojiImage, int(MWidth), int(MWidth), imaging.Linear)

			dc.DrawImageAnchored(dstImageFit, int(x), int(y), 1, 0.5)

			emojiIndex++

			// draw debug line
			// dc.DrawLine(x-MWidth, y-MWidth/2, x, y+MWidth/2)
			// dc.SetLineWidth(1)
			// dc.SetRGB(0, 0, 1)
			// dc.Stroke()

		} else {
			if char == 65039 || char == 65038 { // ufe0f, ufe0e; Variant selectors (ignore)
				continue
			}
			runeWidth, _ := dc.MeasureString(string(char))
			dc.DrawStringAnchored(string(char), x, y, 0, 0.5)
			x += runeWidth
		}
	}

	// Draw debug underline
	// dc.SetRGB(1, 0, 0)
	// dc.DrawLine(startX, y, x, y)
	// dc.Stroke()

	fmt.Println("Done in", time.Since(startTime))

	fmt.Println("Saving to out.png")

	if err := dc.SavePNG("out.png"); err != nil {
		panic(err)
	}

}
