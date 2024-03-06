package main

import (
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	prog "github.com/gosuri/uiprogress"
)

type entry struct {
	url  string
	name string
}

func main() {
	if len(os.Args) < 2 {
		println("Usage: " + os.Args[0] + " https://<anime>.wbijam.pl <dir>")
		os.Exit(1)
	}

	mainUrl := os.Args[1]
	dir := os.Args[2]

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("Katalog nie istnieje, tworzę...")
		os.Mkdir(dir, fs.ModePerm)
	}

	choices, err := getSeasons(mainUrl)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(choices))

	data, err := p.Run()

	if err != nil {
		fmt.Printf("Jakiś błąd: %v", err)
		os.Exit(1)
	}

	chosenSeries := make([]entry, 0)

	for k := range data.(model).selected {
		chosenSeries = append(chosenSeries, choices[k])
	}

	if len(chosenSeries) == 0 {
		fmt.Println("Nie wybrano żadnego sezonu.")
		os.Exit(1)
	}

	startTime := time.Now()

	for _, series := range chosenSeries {

		prog.Start()
		var wg sync.WaitGroup

		eps, err := getEpisodes(series.url)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, ep := range eps {
			wg.Add(1)

			go func(ep animeEp) {
				defer wg.Done()

				err = ep.Download(dir, len(eps))

				if err != nil {
					fmt.Printf("Nie udało się pobrać odcinka, error: %v", err)
				}
			}(ep)
		}

		wg.Wait()

		prog.Stop()
	}

	fmt.Println("Zajęło: " + time.Now().Sub(startTime).String())
}
