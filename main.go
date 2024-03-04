package main

import (
	"fmt"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	prog "github.com/gosuri/uiprogress"
)

type entry struct {
	url  string
	name string
}

func main() {
	if len(os.Args) < 2 {
		println("Usage: " + os.Args[0] + " https://<anime>.wbijam.pl")
		os.Exit(1)
	}

	mainUrl := os.Args[1]
	// dir := os.Args[2]

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

	for _, series := range chosenSeries {

		prog.Start()
		var wg sync.WaitGroup

		eps, err := getEpisodes(series.url)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, ep := range eps {
			// bar := prog.AddBar(100).AppendCompleted().PrependElapsed()

			wg.Add(1)

			// go func(ep animeEp) {
			// 	defer wg.Done()
			if ep.GetMediaURL() == "" {
				fmt.Println("Nie udało się pobrać linku do odcinka")
			}
			// ep.Download(dir, len(eps), bar)
			// }(ep)

			break
		}

		wg.Wait()

		prog.Stop()
	}
}
