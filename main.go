package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Enhanced data structures
type DataItem struct {
	ID            int               `json:"id"`
	Text          string            `json:"text"`
	Category      string            `json:"category"`
	Tags          []string          `json:"tags"`
	Label         string            `json:"label"`
	Confidence    float64           `json:"confidence"`
	UserVerified  bool             `json:"user_verified"`
	ModelPreds    map[string]float64 `json:"model_predictions"`
	LastUpdated   time.Time         `json:"last_updated"`
	Version       int               `json:"version"`
	VerifiedBy    string           `json:"verified_by"`
	ReviewStatus  string           `json:"review_status"`
	History       []ChangeRecord    `json:"history"`
}

type ChangeRecord struct {
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"user"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
}

type DatasetMetadata struct {
	Name           string    `json:"name"`
	Version        int       `json:"version"`
	LastModified   time.Time `json:"last_modified"`
	TotalItems     int       `json:"total_items"`
	VerifiedItems  int       `json:"verified_items"`
	Labels         []string  `json:"labels"`
	Categories     []string  `json:"categories"`
	Tags           []string  `json:"tags"`
}

type MetricsData struct {
	Accuracy        float64            `json:"accuracy"`
	F1Score         float64            `json:"f1_score"`
	DatasetSize     int                `json:"dataset_size"`
	VerifiedPct     float64            `json:"verified_percentage"`
	LabelDistribution map[string]int    `json:"label_distribution"`
	QualityScore    float64            `json:"quality_score"`
	BiasMetrics     map[string]float64 `json:"bias_metrics"`
}

// DataManager handles all data operations
type DataManager struct {
	Dataset     []DataItem
	Metadata    DatasetMetadata
	Metrics     MetricsData
	BackupPath  string
	CurrentUser string
}

func NewDataManager() *DataManager {
	return &DataManager{
		Dataset:    make([]DataItem, 0),
		BackupPath: "backups/",
		Metadata: DatasetMetadata{
			Version:      1,
			LastModified: time.Now(),
		},
	}
}

// Data management methods
func (dm *DataManager) ImportCSV(reader io.Reader) error {
	csvReader := csv.NewReader(reader)
	headers, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("error reading CSV headers: %v", err)
	}

	// Map headers to struct fields
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Read data rows
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading CSV row: %v", err)
		}

		item := DataItem{
			ID:          len(dm.Dataset) + 1,
			LastUpdated: time.Now(),
			ModelPreds:  make(map[string]float64),
			History:     make([]ChangeRecord, 0),
		}

		// Map CSV fields to struct
		for header, idx := range headerMap {
			switch header {
			case "text":
				item.Text = record[idx]
			case "category":
				item.Category = record[idx]
			case "label":
				item.Label = record[idx]
			case "tags":
				item.Tags = strings.Split(record[idx], ",")
			}
		}

		dm.Dataset = append(dm.Dataset, item)
	}

	dm.UpdateMetadata()
	return nil
}

func (dm *DataManager) ExportJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(struct {
		Metadata DatasetMetadata `json:"metadata"`
		Data     []DataItem      `json:"data"`
	}{
		Metadata: dm.Metadata,
		Data:     dm.Dataset,
	})
}

func (dm *DataManager) CreateBackup() error {
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(dm.BackupPath, fmt.Sprintf("backup_%s.json", timestamp))

	if err := os.MkdirAll(dm.BackupPath, 0755); err != nil {
		return fmt.Errorf("error creating backup directory: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating backup file: %v", err)
	}
	defer file.Close()

	return dm.ExportJSON(file)
}

func (dm *DataManager) UpdateItem(index int, updates map[string]interface{}) {
	item := &dm.Dataset[index]
	oldItem := *item

	for field, value := range updates {
		switch field {
		case "label":
			item.Label = value.(string)
		case "tags":
			item.Tags = value.([]string)
		case "category":
			item.Category = value.(string)
		}

		// Record change
		item.History = append(item.History, ChangeRecord{
			Timestamp: time.Now(),
			User:      dm.CurrentUser,
			Field:     field,
			OldValue:  fmt.Sprintf("%v", reflect.ValueOf(oldItem).FieldByName(field).Interface()),
			NewValue:  fmt.Sprintf("%v", value),
		})
	}

	item.LastUpdated = time.Now()
	item.Version++
	dm.UpdateMetadata()
}

func (dm *DataManager) UpdateMetadata() {
	dm.Metadata.LastModified = time.Now()
	dm.Metadata.TotalItems = len(dm.Dataset)

	// Count verified items
	verified := 0
	labelCounts := make(map[string]int)
	categories := make(map[string]bool)
	tags := make(map[string]bool)

	for _, item := range dm.Dataset {
		if item.UserVerified {
			verified++
		}
		labelCounts[item.Label]++
		categories[item.Category] = true
		for _, tag := range item.Tags {
			tags[tag] = true
		}
	}

	dm.Metadata.VerifiedItems = verified
	dm.Metadata.Labels = mapKeys(labelCounts)
	dm.Metadata.Categories = mapKeys(categories)
	dm.Metadata.Tags = mapKeys(tags)

	// Update metrics
	dm.UpdateMetrics(labelCounts)
}

func (dm *DataManager) UpdateMetrics(labelCounts map[string]int) {
	dm.Metrics.DatasetSize = len(dm.Dataset)
	dm.Metrics.VerifiedPct = float64(dm.Metadata.VerifiedItems) / float64(dm.Metadata.TotalItems) * 100
	dm.Metrics.LabelDistribution = labelCounts

	// Calculate quality score based on verification percentage and label distribution
	distributionScore := calculateDistributionScore(labelCounts)
	dm.Metrics.QualityScore = (dm.Metrics.VerifiedPct + distributionScore) / 2

	// Calculate basic bias metrics
	dm.Metrics.BiasMetrics = calculateBiasMetrics(dm.Dataset)
}

// UI Component Creation
func createAnalysisTab(dm *DataManager) *fyne.Container {
	// Create distribution chart
	chart := canvas.NewRectangle(theme.PrimaryColor())
	chart.Resize(fyne.NewSize(400, 200))

	// Create metrics display
	metricsText := binding.NewString()
	metricsLabel := widget.NewLabelWithData(metricsText)

	// Create quality indicators
	qualityProgress := widget.NewProgressBar()
	biasAlert := widget.NewLabel("")

	// Update functions
	updateMetricsDisplay := func() {
		metrics := dm.Metrics
		text := fmt.Sprintf(
			"Dataset Quality Score: %.2f%%\n"+
				"Verified Items: %d/%d (%.1f%%)\n"+
				"Label Distribution:\n",
			metrics.QualityScore,
			dm.Metadata.VerifiedItems,
			metrics.DatasetSize,
			metrics.VerifiedPct,
		)

		for label, count := range metrics.LabelDistribution {
			text += fmt.Sprintf("  %s: %d (%.1f%%)\n", 
				label, 
				count, 
				float64(count)/float64(metrics.DatasetSize)*100,
			)
		}

		metricsText.Set(text)
		qualityProgress.SetValue(metrics.QualityScore / 100)

		// Update bias alert if significant bias detected
		if bias := detectSignificantBias(metrics.BiasMetrics); bias != "" {
			biasAlert.SetText("⚠️ Potential bias detected: " + bias)
		} else {
			biasAlert.SetText("")
		}
	}

	// Create analysis controls
	controls := container.NewVBox(
		widget.NewButton("Refresh Analysis", func() {
			dm.UpdateMetadata()
			updateMetricsDisplay()
		}),
		widget.NewButton("Export Report", func() {
			exportAnalysisReport(dm)
		}),
	)

	// Initial update
	updateMetricsDisplay()

	return container.NewVBox(
		widget.NewLabel("Dataset Analysis"),
		chart,
		metricsLabel,
		container.NewHBox(
			widget.NewLabel("Quality Score:"),
			qualityProgress,
		),
		biasAlert,
		controls,
	)
}

// Helper functions
func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
	})
	return keys
}

func calculateDistributionScore(counts map[string]int) float64 {
	if len(counts) == 0 {
		return 0
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	expected := float64(total) / float64(len(counts))
	var deviation float64

	for _, count := range counts {
		diff := float64(count) - expected
		deviation += (diff * diff)
	}

	deviation = deviation / float64(len(counts))
	maxDeviation := expected * expected

	return (1 - (deviation / maxDeviation)) * 100
}

func calculateBiasMetrics(dataset []DataItem) map[string]float64 {
	metrics := make(map[string]float64)
	
	// Example bias checks (expand based on your needs)
	labelCounts := make(map[string]int)
	textLengths := make(map[string]float64)
	
	for _, item := range dataset {
		labelCounts[item.Label]++
		textLengths[item.Label] += float64(len(item.Text))
	}

	// Check label distribution bias
	total := len(dataset)
	expectedPct := 1.0 / float64(len(labelCounts))
	
	var distributionBias float64
	for label, count := range labelCounts {
		actualPct := float64(count) / float64(total)
		distributionBias += math.Abs(actualPct - expectedPct)
		
		// Check average text length bias
		if count > 0 {
			avgLength := textLengths[label] / float64(count)
			metrics["text_length_"+label] = avgLength
		}
	}
	
	metrics["distribution_bias"] = distributionBias / float64(len(labelCounts))
	return metrics
}

func detectSignificantBias(metrics map[string]float64) string {
	// Check for significant distribution bias
	if metrics["distribution_bias"] > 0.3 {
		return "Significant imbalance in label distribution"
	}

	// Check for text length bias
	var lengths []float64
	for key, value := range metrics {
		if strings.HasPrefix(key, "text_length_") {
			lengths = append(lengths, value)
		}
	}

	if len(lengths) > 1 {
		min, max := minMax(lengths)
		if max/min > 2 {
			return "Large variation in text lengths between classes"
		}
	}

	return ""
}

func minMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}
