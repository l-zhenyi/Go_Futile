package main

import (
	"image"
	"image/color"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	currentNode     *storyNode
	startNode       *storyNode
	choiceAreas     []ChoiceArea
	quitButton      ChoiceArea
	displayedImage  *ebiten.Image
	lastClickTime   time.Time // Track when last click happened
	mouseWasPressed bool      // Track previous mouse state
	screenWidth     int
	screenHeight    int
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if g.displayedImage != nil {
		imgW, imgH := g.displayedImage.Size()

		maxW, maxH := float64(g.screenWidth), 240.0
		startY := 40.0

		scaleX := maxW / float64(imgW)
		scaleY := maxH / float64(imgH)

		scale := scaleX
		if scaleY < scaleX {
			scale = scaleY
		}

		finalW := float64(imgW) * scale
		// finalH := float64(imgH) * scale

		startX := (maxW - finalW) / 2

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(startX, startY)
		screen.DrawImage(g.displayedImage, opts)
	}

	// Draw quit button
	quitText := "[Quit]"
	quitX := g.screenWidth - 80
	quitY := 10
	ebitenutil.DebugPrintAt(screen, quitText, quitX, quitY)
	g.quitButton = ChoiceArea{x: quitX, y: quitY, w: 70, h: 20, index: -1}

	// Draw story text, wrapped
	lines := WrapText(g.currentNode.text, basicfont.Face7x13, g.screenWidth-40) // Subtract some padding
	textY := 300
	for _, line := range lines {
		textBounds := text.BoundString(basicfont.Face7x13, line)
		textWidth := (textBounds.Max.X - textBounds.Min.X)
		textX := (g.screenWidth - textWidth) / 2 // Center each line

		text.Draw(screen, line, basicfont.Face7x13, textX, textY, color.White)
		textY += 20 // Adjust for line height
	}

	// Draw choices below story text
	g.choiceAreas = nil
	yStart := textY + 20 // Start below the last line of text
	for i, c := range g.currentNode.choices {
		choiceText := c.cmd + ": " + c.description
		y := yStart + 20*i

		textBounds := text.BoundString(basicfont.Face7x13, choiceText)
		choiceTextWidth := (textBounds.Max.X - textBounds.Min.X)
		choiceX := (g.screenWidth - choiceTextWidth) / 2

		text.Draw(screen, choiceText, basicfont.Face7x13, choiceX, y, color.White)

		g.choiceAreas = append(g.choiceAreas, ChoiceArea{
			x:     choiceX,
			y:     y - 13, // approximate height adjustment for mouse area
			w:     choiceTextWidth,
			h:     16, // approximate height
			index: i,
		})
	}

	if len(g.currentNode.choices) == 0 {
		ebitenutil.DebugPrintAt(screen, "--- The End ---", 20, 400)
		// Add restart option
		restartText := "[Restart]"
		restartX := 20
		restartY := 420
		ebitenutil.DebugPrintAt(screen, restartText, restartX, restartY)
		g.choiceAreas = append(g.choiceAreas, ChoiceArea{
			x:     restartX,
			y:     restartY - 13,
			w:     70,
			h:     16,
			index: -2, // Special index for restart
		})
	}
}

func (g *Game) Update() error {
	// Get current mouse state
	mouseIsPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	x, y := ebiten.CursorPosition()

	// Check for click RELEASE (not press)
	justReleased := g.mouseWasPressed && !mouseIsPressed
	g.mouseWasPressed = mouseIsPressed

	// Only process clicks if:
	// 1. The mouse was just released (not pressed)
	// 2. At least 200ms since last click (debounce)
	if justReleased && time.Since(g.lastClickTime) > 200*time.Millisecond {
		g.lastClickTime = time.Now()

		// Check quit button
		if x >= g.quitButton.x && x <= g.quitButton.x+g.quitButton.w &&
			y >= g.quitButton.y && y <= g.quitButton.y+g.quitButton.h {
			os.Exit(0)
		}

		// Check choices
		for _, area := range g.choiceAreas {
			if x >= area.x && x <= area.x+area.w &&
				y >= area.y && y <= area.y+area.h {

				if area.index == -2 { // Restart
					g.currentNode = g.startNode
					g.displayedImage = g.startNode.image
					return nil
				}

				g.currentNode = g.currentNode.choices[area.index].nextNode
				if g.currentNode.image != nil {
					g.displayedImage = g.currentNode.image
				} else {
					g.displayedImage = nil
				}
				return nil
			}
		}
	}

	// Cursor handling
	hovering := false
	for _, area := range g.choiceAreas {
		if x >= area.x && x <= area.x+area.w &&
			y >= area.y && y <= area.y+area.y+area.h {
			hovering = true
			break
		}
	}

	if hovering {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}

	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenWidth = outsideWidth
	g.screenHeight = outsideHeight
	return g.screenWidth, g.screenHeight
}

type choice struct {
	cmd         string
	description string
	nextNode    *storyNode
}

type ChoiceArea struct {
	x, y, w, h int
	index      int
}

type storyNode struct {
	text    string
	choices []*choice
	image   *ebiten.Image
}

func loadImage(path string) *ebiten.Image {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	return ebiten.NewImageFromImage(img)
}

func (node *storyNode) addChoice(cmd string, description string, nextNode *storyNode) {
	choice := &choice{cmd, description, nextNode}
	node.choices = append(node.choices, choice)
}

// WrapText wraps the input string to fit within maxWidth in pixels using the given font face.
func WrapText(text string, face font.Face, maxWidth int) []string {
	words := strings.Fields(text)
	lines := []string{}
	if len(words) == 0 {
		return lines
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		testLine := currentLine + " " + word
		w := textWidth(testLine, face)
		if w > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}
	lines = append(lines, currentLine)
	return lines
}

// Helper to measure text width
func textWidth(s string, face font.Face) int {
	width := 0
	for _, r := range s {
		a, _ := face.GlyphAdvance(r)
		width += a.Round()
	}
	return width
}

func main() {
	dark_moon := loadImage("./assets/dark_moon.png")
	red_moon := loadImage("./assets/red_moon.png")
	trees := loadImage("./assets/trees.png")

	start := &storyNode{text: `When you next open your eyes, you are standing on the docks of a crowded harbour. 
	Small boats mill about, bobbing in the ocean. The incessant chatter of guides haggling with potential passengers grates against your ears.`, image: dark_moon}

	// Initialize game with both current and start nodes
	game := &Game{
		currentNode:    start,
		startNode:      start, // Store the starting node
		displayedImage: start.image,
		screenWidth:    640, // Initialize screenWidth
		screenHeight:   480, // Initialize screenHeight
	}

	look := &storyNode{text: `As if by instinct, you are drawn to a boat not too far in the distance.
	The guide, dressed in a shabby blue coat, is watching you.`}
	boat := &storyNode{text: `You walk over but before you can open your mouth, someone jostles you 
	and you stumble. He grabs your arm and helps you board.`}
	turn := &storyNode{text: "You frantically look around for another guide but they are busy."}
	look.addChoice("Yes", "Go with him as your guide", boat)
	look.addChoice("No", "Turn away", turn)
	start.addChoice("Look around", "You try to find another guide", look)

	turn.addChoice("Look back", "He is still watching", start)
	silence_1 := &storyNode{text: `The boat pulls away from the dock rather quickly. 
	The both of you journey in silence for a while.`, image: trees}
	price := &storyNode{text: "\"Don't bother. Sit down.\""}
	fine := &storyNode{text: "\"It's fine.\""}
	boat.addChoice("You don't trust him", "What's your price?", price)
	boat.addChoice("You are grateful", "Thank you", fine)

	price.addChoice("Sit in silence", "He starts rowing", silence_1)
	fine.addChoice("Sit in silence", "He starts rowing", silence_1)
	dead := &storyNode{text: `"Yes. And no." You realise he's punting rather than rowing-using a 
	pole to push against the riverbed. "You're being asked to resettle in an
	existence of not quite living but you're more than half alive."`}
	silence_1.addChoice("Muster up the courage", "\"Am I dead?\"", dead)

	lean := &storyNode{text: "\"Don't lean over,\" he warns. You pull your head back, feeling guilty for some reason. \"The souls ocean is bottomless.\""}
	dead.addChoice("Accept this", "You are oddly calm", lean)

	no_soul := &storyNode{text: `"No," he immediately replies. He moves around so easily on the boat,
	it's hard to imagine him on land. "I don't have a soul."`}
	depths := &storyNode{text: `"Yes." He looks away from his incessant punting and gives you a once over.
	"It will drag down anyone with a soul to the very depths."`}
	bottomless := &storyNode{text: `"Yes." He propels the boat forward by pushing off the riverbed once more.
	"Doesn't make sense but trust me."`}
	lean.addChoice("Ask", "\"Aren't you afraid of falling in?\"", no_soul)
	lean.addChoice("Ask", "\"The souls ocean?\"", depths)
	lean.addChoice("Ask", "\"Bottomless?\"", bottomless)

	silence_2 := &storyNode{text: `At your silence, he turns around. 
	The wind must be chilly but he stands against it like it's a passing breeze. 
	He holds up his pole and flicks his wrist, splashing droplets of water into your face.
	Remembering his warning about souls, your arms come up to protect your face. The droplets fall on your 
	skin harmlessly. You stare at him incredulously.`}
	bottomless.addChoice("Speechless", "Try to enjoy the ride", silence_2)
	depths.addChoice("Speechless", "Try to enjoy the ride", silence_2)
	no_soul.addChoice("Speechless", "Try to enjoy the ride", silence_2)

	grace := &storyNode{text: `"I'm your guide, you know," he says, as if reading your mind. 
	"It's my utmost duty to keep you safe." It takes some time before you pull away completely from the docks. 
	Before you know it, there are no other boats in sight, only the dim outline of the distant shore.`}
	yell := &storyNode{text: `You open your mouth but for some strange reason, 
	you can't bring yourself to scold him. Perplexed, you lean back but don't hide your annoyance.`}
	yell.addChoice("You're not pleased", "You let it go, against your will", grace)
	joke := &storyNode{text: `"No." He turns back to the front. 
	"The lack of oxygen from falling in and drowning will get you. The water itself is harmless."`}
	joke.addChoice("You're not pleased", "You let it go, against your will", grace)

	silence_2.addChoice("You're angry", "Yell at him", yell)
	silence_2.addChoice("Ask", "\"Was that bottomless ocean thing a joke?\"", joke)
	silence_2.addChoice("You're give him grace", "Let it be", grace)

	who := &storyNode{text: `You're about to ask him the question when there's a 
	low groan across the waters. Your guide tenses, in the practiced way when people don't 
	want others to panic.`}

	outrun := &storyNode{text: `"Outrun a monster." Immediately afterwards, a 
	dangerous torrent swirls into a whirlpool. You look back and see a monster tower over the waves.
	"There's no need to worry," he says, in a voice that definitely suggests you should worry.
	`}
	swamp := &storyNode{text: `He deftly manoeuvres the boat into a swamp with low-hanging vines. 
	This only makes the beast speed towards the both of you at new, unfounded speeds.`,
		image: red_moon}
	outrun.addChoice("Believe him", "Don't worry", swamp)
	outrun.addChoice("Don't believe him", "Worrry", swamp)

	catch := &storyNode{text: `"I'm not it's favourite patron."`}
	auditor := &storyNode{text: `"The auditor. For lost and found."`}
	what := &storyNode{text: `He doesn't reply at first. He keeps punting and lookng over his shoulder.
	You hold on to the side of the boat until the wood splinters cut your palms and fingers. From behind,
	the monster keeps drawing closer.`}
	auditor.addChoice("Ask", "\"What?\"", what)
	catch.addChoice("Ask", "\"What?\"", what)

	name := &storyNode{text: `He says your name. It would have completely slipped past you except he becomes 
	utterly still. He meets your eyes like a soldier being told to go on the frontlines.`}
	what.addChoice("Ask", "Hey?", name)

	memory := &storyNode{text: `The memories tear down the mental walls in your mind.
	You remember him now. Blood on his hands, crying over your body and begging to 
	see you one last time. They granted his wish. Each time, he guides you to the realm where you 
	start your new life. He waits to see you between lifetimes and loses you, again and again.`}
	engulf := &storyNode{text: `The creature arrives in the swamp but rather than the brutal impact, horror 
	and gore, a cloud of soft fog so fierce and grey engulfs everything whole. 
	His name eludes you. The fog swallows him up and the traces of your life in this realm are fading away.`}
	memory.addChoice("There is no choice", "There is only one way this ends", engulf)
	end := &storyNode{text: `He raises his hand in farewell and the cycle begins again.`}
	engulf.addChoice("Remember", "Say his name", end)
	engulf.addChoice("Try to be kind", "Don't say his name", end)

	name.addChoice("You remember", "Get your answers", memory)
	name.addChoice("Block it out", "Your mind doesn't want to remember", memory)

	swamp.addChoice("Ask", "\"Why is it chasing us?\"", catch)
	swamp.addChoice("Ask", "\"What is that?\"", auditor)

	nothing := &storyNode{text: `"Nothing-" He exhales defeatedly, catching himself, then says, 
	"Something is happening but it's fine. I've done this before."`}
	nothing.addChoice("Ask", "\"Done what?\"", outrun)

	who.addChoice("Ask", "\"What's wrong?\"", nothing)

	going := &storyNode{text: `"To the place where new things begin."`}
	just_you := &storyNode{text: `"Just you. I'll be staying for the next passenger."`}
	grace.addChoice("Ask", "\"Where are we going?\"", going)
	going.addChoice("Ask", "\"The both of us?\"", just_you)
	just_you.addChoice("Ask", "\"Who are you?\"", who)

	accept := &storyNode{text: `"I was asked and accepted it."`}
	grace.addChoice("Ask", "How did you get this job?", accept)
	accept.addChoice("Ask", "\"Who are you?\"", who)

	time := &storyNode{text: `"Time is difficult here. Around 80 years, I think."`}
	bored := &storyNode{text: `"Not at all."`}
	grace.addChoice("Ask", "\"How long have you been doing this for?\"", time)
	time.addChoice("Ask", "\"80 years? Don't you ever get bored?\"", bored)
	bored.addChoice("Ask", "\"Who are you?\"", who)

	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("But I have to try")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
