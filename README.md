# Fake Insights Generator "FIG"

The goal of this tool is to build an appealing and coherent dashboard experience for our flagship demo(s) by faking insights events in a controlled manner.

## Features

- Generate search traffic, **click and conversion events** from a provided **list of search terms**.
- Target specific percentages of **click-trough rate** and **conversion rate**, globally and per search term.
- Target specific **click positions**, globally and per search term.
- **A/B tests**: Target specific percentages of click-trough rate and conversion rate for a given variant of a running A/B test.
- **Dynamic Synomyns**: Trigger synomyns suggestion for a given search term.

## Usage

### Prerequisites

[Go](https://golang.org/doc/install) > 1.16

### Installation

```bash
git clone git@github.com:algolia/fake-insights-generator.git
```

```bash
go mod tidy
```

### Run

```
go run cmd/main.go events --help
```

1. All flags with default values can be omitted.
2. All flags can be set in a configuration file (`config.yml`) instead of passing them as arguments.


```bash
Generate analytics events

Usage:
  fake-insights-generator events [flags]

Flags:
      --ab-test-variant-ctr float   A/B Test: How much CTR +% for the selected variant (default 20)
      --ab-test-variant-cvr float   A/B Test: How much CTR +% for the selected variant (default 20)
      --ab-test-variant-id int      A/B Test: ID of the variant to favorize
      --api-key string              Algolia API key
      --app-id string               Algolia application ID
      --click-through-rate float    click through rate (default 20)
      --conversion-rate float       conversion rate (default 10)
      --dry-run                     if false, events will not be sent and analytics will be disabled on search queries
  -h, --help                        help for events
      --hits-per-page int           number of hits per page (default 20)
      --index-name string           Algolia index name
      --search-terms string         searches terms file (default "search-terms.csv")
      --searches-per-user int       number of searches per user (default 5)
      --user-tags string            users tags file (default "user-tags.json")
      --users int                   number of users (default 100)
```


