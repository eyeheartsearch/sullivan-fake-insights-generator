package recommend

import (
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/algolia/fake-insights-generator/pkg/iostreams"
	"github.com/go-gota/gota/dataframe"
	"github.com/google/uuid"
)

type Config struct {
	IO *iostreams.IOStreams

	SearchIndex    *search.Index
	InsightsClient *insights.Client
}

type Recommend struct {
	FacetName string
	FBT       map[string][]string
}

func LoadRecommendConfig(config *Config, filePath string) (*Recommend, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var recommend Recommend
	err = json.Unmarshal(bytes, &recommend)
	if err != nil {
		return nil, err
	}
	return &recommend, nil
}

// randomDate returns a random date between min and max
func randomDate(start, end time.Time) time.Time {
	return time.Unix(rand.Int63n(end.Unix()-start.Unix())+start.Unix(), 0)
}

func LoadRecords(cfg *Config) (*dataframe.DataFrame, error) {
	res, err := cfg.SearchIndex.BrowseObjects()
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0)
	for {
		obj, err := res.Next()
		if err != nil {
			break
		}
		objM := obj.(map[string]interface{})
		objM["category_page_id"] = objM["category_page_id"].([]interface{})[len(objM["category_page_id"].([]interface{}))-1]
		items = append(items, obj.(map[string]interface{}))
	}
	df := dataframe.LoadMaps(items)
	return &df, nil
}

func GetRandomObjectIDFromCategory(df *dataframe.DataFrame, r *Recommend, facetName string, category string) string {
	dfFiltered := df.Filter(dataframe.F{
		Colname:    r.FacetName,
		Comparator: "==",
		Comparando: category,
	})
	objectIDs := dfFiltered.Col("objectID").Records()
	if len(objectIDs) == 0 {
		return ""
	}
	return objectIDs[rand.Intn(len(objectIDs))]
}

func Run(config *Config) error {
	r, error := LoadRecommendConfig(config, "recommend.json")
	if error != nil {
		return error
	}

	df, error := LoadRecords(config)
	if error != nil {
		return error
	}

	similarClicks := make(map[string][]string)
	for _, category := range df.Col(r.FacetName).Records() {
		// fmt.Println(category)
		// Similar items clicks (from the same category)
		similarClicks[category] = append(similarClicks[category], uuid.NewString())
	}

	clicksList := make([]insights.Event, 0)
	conversionsList := make([]insights.Event, 0)
	daysAgo := time.Now().Add(-time.Hour * 24 * 90)

	for _, item := range df.Maps() {
		category := item[r.FacetName].(string)
		// 15 clicks per objectID
		for i := 0; i < 15; i++ {
			clickUUID := similarClicks[category][rand.Intn(len(similarClicks[category]))]
			clicksList = append(clicksList, insights.Event{
				UserToken: clickUUID,
				Index:     config.SearchIndex.GetName(),
				ObjectIDs: []string{item["objectID"].(string)},
				Timestamp: randomDate(daysAgo, time.Now()),
				EventType: "click",
				EventName: "click",
			})
		}

		for i := 0; i < 50; i++ {
			conversionUUID := uuid.NewString()
			// Main conversion
			conversionsList = append(conversionsList, insights.Event{
				UserToken: conversionUUID,
				Index:     config.SearchIndex.GetName(),
				ObjectIDs: []string{item["objectID"].(string)},
				Timestamp: randomDate(daysAgo, time.Now()),
				EventType: "conversion",
				EventName: "conversion",
			})
			// FBT conversion, one objectID per FBT category
			for _, FBTCat := range r.FBT[category] {
				objectID := GetRandomObjectIDFromCategory(df, r, r.FacetName, FBTCat)
				if objectID == "" {
					continue
				}
				conversionsList = append(conversionsList, insights.Event{
					UserToken: conversionUUID,
					Index:     config.SearchIndex.GetName(),
					ObjectIDs: []string{objectID},
					Timestamp: randomDate(daysAgo, time.Now()),
					EventType: "conversion",
					EventName: "conversion",
				})
			}
		}
	}

	file, err := os.OpenFile("./events-similar.csv", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	csvWriter := csv.NewWriter(file)
	csvWriter.Write([]string{"userToken", "timestamp", "objectID", "eventType", "eventName"})
	for _, event := range clicksList {
		csvWriter.Write([]string{
			event.UserToken,
			event.Timestamp.Format("2006-01-02T15:04:05Z"),
			event.ObjectIDs[0],
			event.EventType,
			event.EventName,
		})
	}
	csvWriter.Flush()

	file, err = os.OpenFile("./events-fbt.csv", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	csvWriter = csv.NewWriter(file)
	csvWriter.Write([]string{"userToken", "timestamp", "objectID", "eventType", "eventName"})
	for _, event := range conversionsList {
		csvWriter.Write([]string{
			event.UserToken,
			event.Timestamp.Format("2006-01-02T15:04:05Z"),
			event.ObjectIDs[0],
			event.EventType,
			event.EventName,
		})
	}
	csvWriter.Flush()

	return nil
}
