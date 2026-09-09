package repository

import "fmt"

type Scenario struct {
	ID              int
	Name            string
	BatteryCapacity int
	CurrentCharge   int
	WorkTime        float64
	Description     string
	ImageURL        string
	VideoURL        string
	Status          string
	Likes           []int
	ScenarioType    string
}

type Repository struct {
	scenarios []Scenario
}

func NewRepository() (*Repository, error) {
	scenarios := []Scenario{
		{
			ID:              1,
			Name:            "Ретро гейминг",
			BatteryCapacity: 7300,
			CurrentCharge:   90,
			WorkTime:        12,
			Description:     "Приятно проведите время за игрой детства, не переживая за автономность телефона",
			ImageURL:        "http://localhost:9002/scenarios/gaming.png",
			VideoURL:        "http://localhost:9002/scenarios/gaming.mp4",
			Status:          "draft",
			Likes:           []int{101, 102},
			ScenarioType:    "гейминг",
		},
		{
			ID:              2,
			Name:            "Серфинг",
			BatteryCapacity: 5000,
			CurrentCharge:   85,
			WorkTime:        8,
			Description:     "Просматривайте любимые сайты без ограничений (кроме ограничений от РКН)",
			ImageURL:        "http://localhost:9002/scenarios/scrolling.png",
			VideoURL:        "http://localhost:9002/scenarios/scrolling.mp4",
			Status:          "published",
			Likes:           []int{104, 105, 106, 107, 108},
			ScenarioType:    "серфинг",
		},
		{
			ID:              3,
			Name:            "Просмотр кино",
			BatteryCapacity: 4200,
			CurrentCharge:   75,
			WorkTime:        6,
			Description:     "Наслаждайтесь фильмами в дороге, на паре, дома, на работе, да и вообще везде",
			ImageURL:        "http://localhost:9002/scenarios/films.png",
			VideoURL:        "http://localhost:9002/scenarios/films.mp4",
			Status:          "published",
			Likes:           []int{109, 110, 111, 112, 113, 114, 115, 116, 117, 118},
			ScenarioType:    "видео",
		},
		{
			ID:              4,
			Name:            "Чтение",
			BatteryCapacity: 6400,
			CurrentCharge:   95,
			WorkTime:        15,
			Description:     "Читайте книги весь день, а еще лучше, грокайте машинное обучение- это весело!!",
			ImageURL:        "http://localhost:9002/scenarios/reading.png",
			VideoURL:        "http://localhost:9002/scenarios/reading.mp4",
			Status:          "published",
			Likes:           []int{101, 119, 120, 121},
			ScenarioType:    "чтение",
		},
		{
			ID:              5,
			Name:            "Фотосъемка",
			BatteryCapacity: 8100,
			CurrentCharge:   80,
			WorkTime:        10,
			Description:     "Снимайте фото без страха разрядиться, вместо этого переживайте за кадр!",
			ImageURL:        "http://localhost:9002/scenarios/photography.png",
			VideoURL:        "http://localhost:9002/scenarios/photography.mp4",
			Status:          "published",
			Likes:           []int{},
			ScenarioType:    "фото",
		},
		{
			ID:              6,
			Name:            "Общение",
			BatteryCapacity: 5500,
			CurrentCharge:   70,
			WorkTime:        7,
			Description:     "Общайтесь сколько вам захочется и не думайте о зарядке аккумулятора",
			ImageURL:        "http://localhost:9002/scenarios/chatting.png",
			VideoURL:        "http://localhost:9002/scenarios/chatting.mp4",
			Status:          "deleted",
			Likes:           []int{103},
			ScenarioType:    "общение",
		},
	}

	return &Repository{scenarios: scenarios}, nil
}

func (r *Repository) GetScenarios() ([]Scenario, error) {
	if len(r.scenarios) == 0 {
		return nil, fmt.Errorf("список сценариев пуст")
	}
	return r.scenarios, nil
}

func (r *Repository) GetScenarioByID(id int) (Scenario, error) {
	for _, s := range r.scenarios {
		if s.ID == id {
			return s, nil
		}
	}
	return Scenario{}, fmt.Errorf("сценарий не найден")
}

func (r *Repository) GetNextScenario(currentID int) (Scenario, error) {
	for i, s := range r.scenarios {
		if s.ID == currentID && i+1 < len(r.scenarios) {
			return r.scenarios[i+1], nil
		}
	}
	if len(r.scenarios) > 0 {
		return r.scenarios[0], nil
	}
	return Scenario{}, fmt.Errorf("нет доступных сценариев")
}

func (r *Repository) GetDraftScenario() (Scenario, error) {
	for _, s := range r.scenarios {
		if s.Status == "draft" {
			return s, nil
		}
	}
	return Scenario{}, fmt.Errorf("черновик не найден")
}

func (r *Repository) GetPublishedScenarios() ([]Scenario, error) {
	var result []Scenario
	for _, s := range r.scenarios {
		if s.Status != "deleted" {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *Repository) GetScenariosByBatteryCapacity(capacity int) ([]Scenario, error) {
	var result []Scenario
	for _, s := range r.scenarios {
		if s.Status != "deleted" && s.BatteryCapacity == capacity {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *Repository) GetLikesCount(id int) int {
	for _, s := range r.scenarios {
		if s.ID == id {
			return len(s.Likes)
		}
	}
	return 0
}
