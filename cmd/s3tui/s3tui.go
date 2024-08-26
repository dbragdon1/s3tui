package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"s3tui/pkg/cache"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)



type model struct {
	cursor				 int
	selectedItem  string 
	currentKey string
	content string
	s3_svc *s3.Client
	Bucket string
	maxKeys int32
	items *[]string
	depth int
	itemCache *cache.ItemCache 
}


func initialModel(s3_svc *s3.Client, path string, itemCache *cache.ItemCache) model {
	return model{
		cursor: 0,
		selectedItem: "",
		currentKey: "",
		items: &[]string{},
		content: "",
		s3_svc: s3_svc,
		maxKeys: 10,
		depth: 0,
		itemCache: itemCache,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) listBuckets() error {
	*m.items = (*m.items)[:0]

	input := &s3.ListBucketsInput{}

	result, wasCached := m.itemCache.Cache[m.currentKey] 
	
	if wasCached {
		*m.items = result.Items
		return nil
	} else {
		resp, err := m.s3_svc.ListBuckets(context.TODO(),input)

		if err != nil {
			return err
		} else {

			for _, bucket := range resp.Buckets {
				*m.items = append(*m.items, *bucket.Name)
			}
			return nil
		}
		}
}

func (m model) ListObjects() error {
	delim := "/"
	maxKeys := int32(10)

	*m.items = (*m.items)[:0]

	input := &s3.ListObjectsV2Input{
		Bucket: &m.Bucket,
		Prefix: &m.currentKey,
		Delimiter: &delim,
		MaxKeys: &maxKeys,
	}

	p := s3.NewListObjectsV2Paginator(m.s3_svc, input, func(o *s3.ListObjectsV2PaginatorOptions) {
		if v := int32(m.maxKeys); v != 0 {
			o.Limit = v
		}
	})

	for p.HasMorePages() {

		page, err := p.NextPage(context.TODO())

		if err != nil {
			return err
		}

		for _, obj := range page.Contents {
			fmt.Printf("Key: %s\n", *obj.Key)

			*m.items = append(*m.items, *obj.Key)
		}
	}
	
	fmt.Printf("Bucket: %s\n", m.Bucket)

	fmt.Printf("Key: %s\n", m.currentKey)

	fmt.Printf("Items: %v\n", *m.items)

	fmt.Printf("Delimiter: %v\n", delim)

	return nil
}

func (m model) View() string {
	var s string
	var listBucketError error
	var listObjectsError error
	var err_s string

	if m.selectedItem == "" {
		s = "Buckets:\n"
		listBucketError = m.listBuckets()
	} else {
		s = "Objects:\n"
		listObjectsError = m.ListObjects()
	} 

	if listBucketError != nil {
		err_s = fmt.Sprintf("Error listing buckets: %v\n", listBucketError)
	} else if listObjectsError != nil {
		err_s  = fmt.Sprintf("Error listing objects: %v\n", listObjectsError)
	} else {
		for i, item := range  *m.items {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			s += fmt.Sprintf("%s %s\n", cursor, item)
		}
	}

	if err_s != "" {
		return wordwrap.String(err_s, 80) 
	} else {
		return s
	}
}

func ParseS3URI(s3_uri string) (string, string) {
	bucket_pattern := regexp.MustCompile(`^s3://([^/]+)/`)

	key_pattern := regexp.MustCompile(`^s3://[^/]+(/.+)$`)

	bucket_matches := bucket_pattern.FindStringSubmatch(s3_uri)

	key_matches := key_pattern.FindStringSubmatch(s3_uri)

	return bucket_matches[1], key_matches[1]
}


func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl-c", "q":
			return m, tea.Quit
		case "j":
			m.cursor++
			if m.cursor >= len(*m.items) {
				m.cursor = len(*m.items) - 1
			}
		case "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "enter", " ":
				m.selectedItem = (*m.items)[m.cursor]
				if m.Bucket == "" {
					m.Bucket = m.selectedItem
				} else {
					m.currentKey = m.selectedItem
				}
		}
	}

	return m, nil
}

func createS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	
	return client, nil
}

func main() {

	s3_client, err := createS3Client()

	if err != nil {
		fmt.Printf("Couldn't authenticate to AWS: %v \n", err)
		os.Exit(1)
	}

	model := initialModel(s3_client, "s3://", cache.NewCache(cache.CacheConfig{}))

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
