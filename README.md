# Fake Insights Generator "FIG"

The goal of this tool is to build an appealing and coherent dashboard experience for our flagship demo(s) by faking insights events in a controlled manner.

## Features

- Generate search traffic, **click and conversion events** from a provided **list of search terms**.
- Target specific percentages of **click-trough rate** and **conversion rate**, globally and per search term.
- Target specific **click positions**, globally and per search term.
- **A/B tests**: Target specific percentages of click-trough rate and conversion rate for a given variant of a running A/B test.
- **Dynamic Synomyns**: Trigger synomyns suggestion for a given search term.

## Installation

1. Download the latest release binary from [GitHub](https://github.com/algolia/fake-insights-generator/releases).
2. Simlink the binary to your PATH (e.g. `ln -s /path/to/fig /usr/local/bin/fig`).
3. Run `fig --help` to see the available options.

## Usage

💡 All command flags with default values can be omitted.

💡 All command flags can be set in a configuration file (ex: [config-sample.yml](./config-sample.yml)) instead of passing them as arguments.

### The configuration files

**[search.json](searches.json)**

This is the main configuration file. It contains the list of search terms to generate events for. Each search term is a JSON object with the following shape:
```json
{
  // The search term itself, mandatory. 
  "term": "men pants",
  // Filters to add to the search terms, optional.
  // Note that each filter value are "weighted".
  // In the exemple below, the first filter value would be choosen two time more often than the second one.
  "filters": {
      "category_page_id": {
        "Women > Bags": 2,
        "Accessories > Women": 1
      }
  },
  // Dynamic synoyms trigger, optional.
  // If present, a second search action will be performed with the following search terms, directly after the main search term search action.
  // If populated, note that the CTR and the CVR will be applied to the synonyms, not to the main search term (the main search term CTR and CVR will be 0).
  "synonyms": ["men trousers"],
  // Click position, optional.
  // If not present, the global click position will be used.
  // Note that this is a "targetted click position". FIG is using a weighted random distribution based on a formula from @guillaume and @shuprelle.
  "click_position": 3,
  // Click through rate, optional.
  // If not present, the global click through rate will be used.
  "click_through_rate": 20,
  // Conversion rate, optional.
  // If not present, the global conversion rate will be used.
  "conversion_rate": 10
}
```

💡 The search terms are picked in a weighted random fashion from the list, the upper the search term is in the file, the more likely it will be picked.

**[event-names.json](events-names.json)**

This is the list of event names to generate. For each event type (click, conversion, view), there is a list of event names to pick from:
```json
{
  // Click event names.
  "click": {
    // The event name are picked in a weighted random way. 8 and 5 below, are the weight.
    "PLP: Open product details": 8,
    "Autocomplete: Open product details": 5,
  },
  // Conversion event names.
  "conversion": {
    "PLP: Add to cart": 8,
    "PLP: Checkout": 5
  },
  // View event names (not used yet in the tool).
  "view": {
    "PLP: Product Viewed": 8,
    "Autocomplete: Product Viewed": 3
  }
}
```

**[user-tags.json](user-tags.json)**

This is the list analytics tags to add to the events. Same as the event names, weighted random style.

**[personas.json](personas.json)**

For personalization, we use a list of personas. Each persona is a JSON object with the following shape:
```json
{
  // The persona description (not used in the tool, but good to have).
  "description": "Woman who likes black dresses (size S)",
  // The userToken
  "token": "mrs-grim",
  // The search terms this persona will search for.
  "terms": ["black dress", "dress"],
  // The filters that will be applied to the search terms (weighted random style again).
  "filters": {
    "category_page_id": {
      "Women > Clothing > Dresses": 10
    }
  }
}
```

All setup? Let's go 👇🏻
```bash
fig events --app-id <app_id> --api-key <api_key> --index-name <index_name> --dry-run
```

The above command should generate events and return the stats.

💡 Note that we added the `--dry-run` flag to the command. This will not actually send the events to the Algolia API (and analytics will be disabled on the searches queries). It's a good way to test the events generation, without sending anything and messing up your analytics dashboard.

```bash
fig events --help
```

### FAQ / Troubleshooting

<details>
<summary>Why is so slow?</summary>
The default delay between each search for each user is `46 seconds`. We setup this delay to avoid triggering unwanted dynamic synonyms. If you don't care about the dynamic synonyms at all, you can set the delay to `0` (or any other value).
</details>

<details>
<summary>I have some weird errors, like `Error doing search: cannot read body: context deadline exceeded`</summary>
It means you are running too many searches at the same time. You should reduce the number of `users` and/or `searches-per-user` to avoid this.
</details>

<details>
<summary>The dynamic synonyms are not working!</summary>
In order for your synonyms to work correctly, you will need to add a common word between the main search term and the synonyms. For example, if you have a main search term "trousers" and a synonym "pants", you will need to add a common word between them, like "men trousers" and "men pants".
</details>
