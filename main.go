package main

import (
	"fmt"
	"math/rand"
	"time"
	"strings"
	"strconv"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"image/color"
)

// DataItem represents a training example
type DataItem struct {
	ID          int
	Text        string
	Category    string
	Tags        []string
	Label       string
	Confidence  float64
	UserVerified bool
	ModelPreds   map[string]float64
	LastUpdated  time.Time
}

// Dataset holds our training data
var dataset = []DataItem{
	{
		ID: 1, 
		Text: "First example item with some interesting properties", 
		Category: "Training",
		Tags: []string{"unverified", "batch1"},
		Label: "positive",
		Confidence: 0.85,
		UserVerified: false,
		ModelPreds: map[string]float64{"positive": 0.85, "negative": 0.15},
		LastUpdated: time.Now(),
	},
	// Add more items...
}

// Training metrics
type MetricsData struct {
	Accuracy    float64
	F1Score     float64
	DatasetSize int
	VerifiedPct float64
}

func main() {
	myApp := app.New()
	window := myApp.NewWindow("ML Training Data Review")

	// Bind data for live updates
	currentIndex := 0
	metrics := binding.BindFloat("accuracy", 0.0)
	trainingStatus := binding.BindString("status", "Ready")

	// Create UI elements
	textDisplay := widget.NewTextGrid()
	idLabel := widget.NewLabel("")
	categoryLabel := widget.NewLabel("")
	tagsLabel := widget.NewLabel("")
	confidenceLabel := widget.NewLabel("")
	
	// Model prediction bars
	predictionBars := make(map[string]*widget.ProgressBar)
	for _, label := range []string{"positive", "negative", "neutral"} {
		predictionBars[label] = widget.NewProgressBar()
	}

	// Quick label buttons
	labelButtons := container.NewHBox(
		widget.NewButton("👍 Positive", func() { setLabel(currentIndex, "positive") }),
		widget.NewButton("👎 Negative", func() { setLabel(currentIndex, "negative") }),
		widget.NewButton("😐 Neutral", func() { setLabel(currentIndex, "neutral") }),
		widget.NewButton("⚠️ Flag for Review", func() { flagForReview(currentIndex) }),
	)

	// Search and filter
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search text or tags...")
	categorySelect := widget.NewSelect([]string{"All", "Training", "Test", "Validation"}, func(s string) {
		filterByCategory(s)
	})

	// Training controls
	trainingControls := container.NewVBox(
		widget.NewButton("Train Quick Model", func() { trainModel("quick") }),
		widget.NewButton("Train Deep Model", func() { trainModel("deep") }),
		widget.NewProgressBar(), // Training progress
		widget.NewLabelWithData(trainingStatus),
	)

	// Stats and metrics
	metricsDisplay := widget.NewTextGrid()
	updateMetrics := func() {
		metrics := calculateMetrics()
		metricsText := fmt.Sprintf(
			"Dataset Metrics:\n"+
				"Total Examples: %d\n"+
				"Verified: %.1f%%\n"+
				"Model Accuracy: %.2f%%\n"+
				"F1 Score: %.2f\n"+
				"Last Training: %s",
			metrics.DatasetSize,
			metrics.VerifiedPct,
			metrics.Accuracy*100,
			metrics.F1Score,
			time.Now().Format("15:04:05"),
		)
		metricsDisplay.SetText(metricsText)
	}

	// Function to update item display
	updateDisplay := func(index int) {
		item := dataset[index]
		textDisplay.SetText(item.Text)
		idLabel.SetText(fmt.Sprintf("ID: %d", item.ID))
		categoryLabel.SetText(fmt.Sprintf("Category: %s", item.Category))
		tagsLabel.SetText(fmt.Sprintf("Tags: %s", strings.Join(item.Tags, ", ")))
		confidenceLabel.SetText(fmt.Sprintf("Confidence: %.2f%%", item.Confidence*100))
		
		// Update prediction bars
		for label, bar := range predictionBars {
			if pred, ok := item.ModelPreds[label]; ok {
				bar.SetValue(pred)
			} else {
				bar.SetValue(0)
			}
		}
		
		updateMetrics()
	}

	// Navigation
	prevButton := widget.NewButton("← Previous", func() {
		if currentIndex > 0 {
			currentIndex--
			updateDisplay(currentIndex)
		}
	})

	nextButton := widget.NewButton("Next →", func() {
		if currentIndex < len(dataset)-1 {
			currentIndex++
			updateDisplay(currentIndex)
		}
	})

	randomButton := widget.NewButton("🎲 Random", func() {
		currentIndex = rand.Intn(len(dataset))
		updateDisplay(currentIndex)
	})

	// Create tabs for different views
	tabs := container.NewAppTabs(
		container.NewTabItem("Review", createReviewTab(
			textDisplay, 
			container.NewVBox(
				idLabel, 
				categoryLabel, 
				tagsLabel,
				confidenceLabel,
				labelButtons,
				container.NewHBox(prevButton, randomButton, nextButton),
			),
			predictionBars,
		)),
		container.NewTabItem("Training", container.NewVBox(
			trainingControls,
			metricsDisplay,
		)),
		container.NewTabItem("Analysis", createAnalysisTab()),
	)

	// Top toolbar
	toolbar := container.NewHBox(
		searchEntry,
		categorySelect,
		widget.NewButton("Export Data", exportData),
		widget.NewButton("Import Data", importData),
	)

	// Main layout
	mainContent := container.NewBorder(
		toolbar, nil, nil, nil,
		tabs,
	)

	// Initial display
	updateDisplay(currentIndex)

	// Set window content and show
	window.SetContent(mainContent)
	window.Resize(fyne.NewSize(1024, 768))
	window.ShowAndRun()
}

// Helper functions (implement these based on your needs)
func setLabel(index int, label string) {
	dataset[index].Label = label
	dataset[index].UserVerified = true
	dataset[index].LastUpdated = time.Now()
}

func flagForReview(index int) {
	dataset[index].Tags = append(dataset[index].Tags, "needs_review")
	dataset[index].LastUpdated = time.Now()
}

func filterByCategory(category string) {
	// Implement filtering logic
}

func trainModel(mode string) {
	// Simulate training
	// In reality, you'd probably want to call your actual ML training code here
	time.Sleep(2 * time.Second)
}

func calculateMetrics() MetricsData {
	verified := 0
	for _, item := range dataset {
		if item.UserVerified {
			verified++
		}
	}
	return MetricsData{
		Accuracy:    0.85, // Replace with actual metrics
		F1Score:     0.83,
		DatasetSize: len(dataset),
		VerifiedPct: float64(verified) / float64(len(dataset)) * 100,
	}
}

func createReviewTab(text *widget.TextGrid, controls fyne.CanvasObject, bars map[string]*widget.ProgressBar) *fyne.Container {
	predictionBox := container.NewVBox()
	for label, bar := range bars {
		predictionBox.Add(widget.NewLabel(label))
		predictionBox.Add(bar)
	}

	return container.NewHSplit(
		text,
		container.NewVBox(
			controls,
			widget.NewCard("Model Predictions", "", predictionBox),
		),
	)
}

func createAnalysisTab() *fyne.Container {
	// Create some mock visualizations
	return container.NewVBox(
		widget.NewLabel("Distribution of Labels"),
		widget.NewProgressBar(), // Mock chart
		widget.NewLabel("Confidence Over Time"),
		widget.NewProgressBar(), // Mock chart
	)
}

func exportData() {
	// Implement export logic
}

func importData() {
	// Implement import logic
}
