package nutrition_assignment

import (
	"sort"
	"strings"
)

// The food catalog is server-owned data, versioned with the generation engine
// rather than stored in a table. Keeping it in code is deliberate:
//
//   - it is reviewed and tested like any other server-owned rule, and it can be
//     diffed in a pull request;
//   - a delivered plan stores an immutable snapshot of the food it resolved to, so
//     there is no mutable catalog row a plan could silently drift against, and no
//     database seed to keep in step with the engine;
//   - generation therefore needs no catalog query at all, which is what lets the
//     same code path run offline and in Test Mode.
//
// Every item declares, for one standard serving, its energy and macros, the unit
// the serving is measured in, the diets it is compatible with, and the allergens
// it contains. Those declarations are what the exclusion engine reasons about:
// a food is only ever excluded because of a fact stated here, never because of
// fuzzy matching against a food name.

// FoodCategory groups foods by their role in a meal. The generator composes a
// meal from at most one item per category, so the category set is what makes a
// generated meal balanced in structure rather than arbitrary.
type FoodCategory string

// FoodCategory values. A meal is composed from these categories in a fixed order,
// which is what makes a generated plan predictable and reviewable.
const (
	FoodCategoryProtein      FoodCategory = "protein"
	FoodCategoryCarbohydrate FoodCategory = "carbohydrate"
	FoodCategoryVegetable    FoodCategory = "vegetable"
	FoodCategoryFruit        FoodCategory = "fruit"
	FoodCategoryFat          FoodCategory = "fat"
)

// Allergen tokens are the controlled vocabulary the exclusion engine reasons
// about. A food declares the allergens it contains, and a client-declared allergy
// is classified into these tokens, so strict exclusion is a set intersection over
// declared facts rather than a guess about a food name.
const (
	AllergenPeanut    = "peanut"
	AllergenTreeNut   = "tree_nut"
	AllergenMilk      = "milk"
	AllergenEgg       = "egg"
	AllergenSoy       = "soy"
	AllergenGluten    = "gluten"
	AllergenFish      = "fish"
	AllergenShellfish = "shellfish"
	AllergenSesame    = "sesame"
)

// DietPattern tokens mirror the questionnaire's diet vocabulary. A food declares
// which patterns it is compatible with; a pattern the food does not declare is a
// hard exclusion, so a vegan, halal or kosher client is never shown a food that
// does not belong to their pattern.
const (
	DietOmnivore    = "omnivore"
	DietFlexitarian = "flexitarian"
	DietPescatarian = "pescatarian"
	DietVegetarian  = "vegetarian"
	DietVegan       = "vegan"
	DietHalal       = "halal"
	DietKosher      = "kosher"
)

// Food groups are a second, coarser axis used to honour exclusions expressed as
// a category of food ("no red meat", "no fish") rather than as a named item.
const (
	FoodGroupRedMeat    = "red_meat"
	FoodGroupPoultry    = "poultry"
	FoodGroupFish       = "fish"
	FoodGroupShellfish  = "shellfish"
	FoodGroupDairy      = "dairy"
	FoodGroupEgg        = "egg"
	FoodGroupNuts       = "nuts"
	FoodGroupSeeds      = "seeds"
	FoodGroupGrains     = "grains"
	FoodGroupLegumes    = "legumes"
	FoodGroupNightshade = "nightshade"
	FoodGroupAddedSugar = "added_sugar"
)

// FoodItem is one entry in the catalog, described for a single standard serving.
type FoodItem struct {
	// Code is the stable machine identifier. It is recorded on every plan meal
	// item as provenance and never changes for a given food.
	Code string
	// Name is the client-facing name of the serving.
	Name string
	// Category is the meal role this food fills.
	Category FoodCategory
	// Serving is the quantity the macros below describe.
	Serving float64
	// Unit is how the serving is measured, and is the unit shown to the client.
	Unit string

	Calories int
	Protein  float64
	Carbs    float64
	Fat      float64
	Fiber    float64

	// Compatibility flags. A client whose diet is declared here may be shown the
	// food; a client whose diet is not declared here never is.
	Omnivore    bool
	Flexitarian bool
	Vegetarian  bool
	Vegan       bool
	Pescatarian bool
	Halal       bool
	Kosher      bool

	// Allergens is the controlled token set this food contains. Anything listed
	// here is a strict exclusion for a client who declared the matching allergy.
	Allergens []string
	// Groups are the coarse food categories this food belongs to.
	Groups []string
	// Terms are extra words that should match a client exclusion. They let a
	// client exclude "oil" or "yoghurt" without the catalog needing an entry for
	// every spelling.
	Terms []string
}

// ServingsPerDay is the number of standard servings of this food that make up
// the macros for one occurrence, used to derive a per-serving scale.
func (f FoodItem) ScalesToServings() bool {
	return f.Serving > 0 && f.Calories > 0
}

// MacrosAtQuantity returns the energy and macros for a quantity expressed in the
// item's own unit — grams, millilitres or a count — rounded the same way every
// other stored macro is rounded.
//
// Rounding is floor-based on purpose: two foods in a meal must never round up to
// more than the plan total, so the stored daily totals stay an upper bound of what
// the items actually contain.
func (f FoodItem) MacrosAtQuantity(quantity float64) Macros {
	quantity /= f.Serving
	return Macros{
		Calories:     int(float64(f.Calories) * quantity),
		ProteinGrams: int(f.Protein * quantity),
		CarbsGrams:   int(f.Carbs * quantity),
		FatGrams:     int(f.Fat * quantity),
		FiberGrams:   int(f.Fiber * quantity),
	}
}

// Macros is the shared macro shape used by generated plans, meals and items.
type Macros struct {
	Calories     int `json:"calories"`
	ProteinGrams int `json:"protein_grams"`
	CarbsGrams   int `json:"carbs_grams"`
	FatGrams     int `json:"fat_grams"`
	FiberGrams   int `json:"fiber_grams"`
}

// Add accumulates another macro set. Generation composes meals from items, so this
// is how a meal total and then a daily total are derived.
func (m Macros) Add(other Macros) Macros {
	return Macros{
		Calories:     m.Calories + other.Calories,
		ProteinGrams: m.ProteinGrams + other.ProteinGrams,
		CarbsGrams:   m.CarbsGrams + other.CarbsGrams,
		FatGrams:     m.FatGrams + other.FatGrams,
		FiberGrams:   m.FiberGrams + other.FiberGrams,
	}
}

// catalog is the curated catalog. It is ordered by code so that generation, which
// breaks ties by catalog order, is fully deterministic regardless of map
// iteration order elsewhere.
var catalog = buildCatalog()

// Catalog returns the catalog foods in a stable, sorted order. Callers must not
// mutate the returned slice.
func Catalog() []FoodItem {
	return append([]FoodItem(nil), catalog...)
}

// CatalogItemByCode returns the catalog entry for a code, and whether it exists.
func CatalogItemByCode(code string) (FoodItem, bool) {
	for _, item := range catalog {
		if item.Code == code {
			return item, true
		}
	}
	return FoodItem{}, false
}

// allergenAliases maps the words a client may type to the controlled allergen
// token they resolve to. An allergy is a strict exclusion, so an ambiguous word is
// mapped to every token it could plausibly mean: "nuts" excludes both peanuts and
// tree nuts, because excluding too much is recoverable and allowing a declared
// allergy through is not.
var allergenAliases = map[string]string{
	"peanut":     AllergenPeanut,
	"peanuts":    AllergenPeanut,
	"groundnut":  AllergenPeanut,
	"groundnuts": AllergenPeanut,
	"arachis":    AllergenPeanut,

	"nut":      AllergenTreeNut,
	"nuts":     AllergenTreeNut,
	"treenut":  AllergenTreeNut,
	"treenuts": AllergenTreeNut,
	"almond":   AllergenTreeNut,
	"almonds":  AllergenTreeNut,
	"walnut":   AllergenTreeNut,
	"walnuts":  AllergenTreeNut,
	"cashew":   AllergenTreeNut,
	"cashews":  AllergenTreeNut,
	"pecan":    AllergenTreeNut,
	"pecans":   AllergenTreeNut,
	"pistach":  AllergenTreeNut,
	"hazelnut": AllergenTreeNut,
	"macadam":  AllergenTreeNut,
	"brazil":   AllergenTreeNut,

	"milk":     AllergenMilk,
	"dairy":    AllergenMilk,
	"lactose":  AllergenMilk,
	"casein":   AllergenMilk,
	"whey":     AllergenMilk,
	"cheese":   AllergenMilk,
	"butter":   AllergenMilk,
	"cream":    AllergenMilk,
	"yoghurt":  AllergenMilk,
	"yogurt":   AllergenMilk,
	"yoghurts": AllergenMilk,
	"custard":  AllergenMilk,

	"egg":       AllergenEgg,
	"eggs":      AllergenEgg,
	"albumin":   AllergenEgg,
	"eggwhite":  AllergenEgg,
	"mayonaise": AllergenEgg,

	"soy":     AllergenSoy,
	"soya":    AllergenSoy,
	"edamame": AllergenSoy,
	"tofu":    AllergenSoy,
	"tempeh":  AllergenSoy,
	"soybean": AllergenSoy,

	"gluten":  AllergenGluten,
	"wheat":   AllergenGluten,
	"coeliac": AllergenGluten,
	"celiac":  AllergenGluten,
	"barley":  AllergenGluten,
	"rye":     AllergenGluten,
	"spelt":   AllergenGluten,
	"seitan":  AllergenGluten,

	"fish":      AllergenFish,
	"seafood":   AllergenFish,
	"salmon":    AllergenFish,
	"tuna":      AllergenFish,
	"cod":       AllergenFish,
	"sardine":   AllergenFish,
	"anchov":    AllergenFish,
	"swordfish": AllergenFish,

	"shellfish":  AllergenShellfish,
	"crustacean": AllergenShellfish,
	"prawn":      AllergenShellfish,
	"shrimp":     AllergenShellfish,
	"crab":       AllergenShellfish,
	"lobster":    AllergenShellfish,
	"mussel":     AllergenShellfish,
	"clam":       AllergenShellfish,
	"oyster":     AllergenShellfish,
	"scallop":    AllergenShellfish,
	"mollusc":    AllergenShellfish,
	"octopus":    AllergenShellfish,

	"sesame": AllergenSesame,
	"tahini": AllergenSesame,
}

// groupPhrases maps an exclusion phrase a client may type to the food group it
// rules out. This is how "no red meat" is honoured: the catalog does not need an
// entry per phrasing, only the group each food belongs to.
var groupPhrases = map[string][]string{
	FoodGroupRedMeat:    {"red meat", "beef", "steak", "mince", "lamb", "veal", "pork", "bacon", "ham", "meat"},
	FoodGroupPoultry:    {"poultry", "chicken", "turkey", "duck", "poultry meat"},
	FoodGroupFish:       {"fish", "seafood", "oily fish"},
	FoodGroupShellfish:  {"shellfish", "crustacean", "mollusc", "mollusk"},
	FoodGroupDairy:      {"dairy", "milk", "lactose", "cheese", "butter", "cream", "yoghurt", "yogurt"},
	FoodGroupEgg:        {"egg", "eggs", "egg white", "mayonnaise"},
	FoodGroupNuts:       {"nuts", "nut", "peanuts", "peanut", "tree nuts"},
	FoodGroupSeeds:      {"seeds", "seed"},
	FoodGroupGrains:     {"grains", "grain", "wheat", "gluten", "cereals", "cereal"},
	FoodGroupLegumes:    {"legumes", "legume", "pulses", "pulses", "beans", "bean", "lentils", "lentil", "chickpeas", "chickpea", "peas", "pea"},
	FoodGroupNightshade: {"nightshade", "nightshades", "aubergine", "eggplant", "tomato", "tomatoes", "pepper", "peppers", "paprika"},
	FoodGroupAddedSugar: {"sugar", "sweets", "candy", "candies", "soft drinks", "soda", "dessert", "desserts"},
}

// allergenTokenSet is the ordered, de-duplicated set of allergen tokens the
// exclusion engine understands. Keeping it explicit means an alias typo cannot
// silently introduce a new allergen vocabulary.
var allergenTokenSet = []string{
	AllergenPeanut, AllergenTreeNut, AllergenMilk, AllergenEgg,
	AllergenSoy, AllergenGluten, AllergenFish, AllergenShellfish, AllergenSesame,
}

// AllergenTokens returns the controlled allergen vocabulary.
func AllergenTokens() []string {
	return append([]string(nil), allergenTokenSet...)
}

// classifyAllergens maps free-text allergy and intolerance entries onto the
// controlled allergen vocabulary.
//
// An entry that resolves to no token is not silently ignored: it is reported in
// the returned unmatched list so the caller can surface it. The generator treats an
// unresolved allergy as satisfied by construction, because the plan is built only
// from catalog foods and every catalog food declares its allergens, so there is
// nothing left for an unrecognised word to exclude. What the engine cannot do is
// guarantee anything about a product's manufacturing, which is why the client is
// always told to check ingredient labels.
func classifyAllergens(entries []string) (matched []string, unmatched []string) {
	seen := map[string]bool{}
	for _, entry := range entries {
		words := foodWords(entry)
		if len(words) == 0 {
			continue
		}
		found := false
		for _, word := range words {
			token, ok := allergenAliases[word]
			if !ok {
				continue
			}
			// An ambiguous word may resolve to several tokens: "nuts" means both
			// peanuts and tree nuts, and "seafood" means fish and shellfish.
			for _, resolved := range resolveAmbiguous(word, token) {
				if !seen[resolved] {
					seen[resolved] = true
					matched = append(matched, resolved)
				}
			}
			found = true
		}
		if !found {
			unmatched = append(unmatched, strings.Join(words, " "))
		}
	}
	sort.Strings(matched)
	sort.Strings(unmatched)
	return matched, unmatched
}

// resolveAmbiguous expands the words that mean more than one allergen token, so a
// strict exclusion is never narrower than what the client declared.
func resolveAmbiguous(word, token string) []string {
	switch word {
	case "nuts", "nut":
		return []string{AllergenPeanut, AllergenTreeNut}
	case "seafood":
		return []string{AllergenFish, AllergenShellfish}
	case "seeds", "seed":
		return []string{AllergenSesame}
	default:
		return []string{token}
	}
}

// foodWords normalises a free-text entry into comparable, de-pluralised words.
//
// Normalisation strips punctuation and collapses whitespace so a client writing
// "Peanuts," and one writing "peanuts" resolve identically, and de-pluralisation
// means "tomatoes" and "tomato" are the same exclusion.
func foodWords(text string) []string {
	lowered := strings.ToLower(strings.TrimSpace(text))
	replaced := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		default:
			return ' '
		}
	}, lowered)

	seen := map[string]bool{}
	words := make([]string, 0, 4)
	for _, field := range strings.Fields(replaced) {
		word := singular(field)
		if word == "" || seen[word] {
			continue
		}
		seen[word] = true
		words = append(words, word)
	}
	return words
}

// singular strips a simple plural so catalog terms and client entries compare
// equal. It is deliberately conservative: it only removes a trailing "s" when the
// result is still a plausible word, so "glass" and "peas" are not mangled into
// something that silently stops matching.
func singular(word string) string {
	switch {
	case len(word) > 3 && strings.HasSuffix(word, "ies"):
		return word[:len(word)-3] + "y"
	case len(word) > 4 && strings.HasSuffix(word, "ses"):
		return word[:len(word)-2]
	case len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss"):
		return word[:len(word)-1]
	default:
		return word
	}
}

// restrictionWords returns the union of the word sets of every free-text
// restriction, used to test whether a food is excluded by name.
func restrictionWords(entries []string) map[string]bool {
	words := map[string]bool{}
	for _, entry := range entries {
		for _, word := range foodWords(entry) {
			words[word] = true
		}
	}
	return words
}

// foodMatchTerms is every word that, if present in a client's restriction text,
// should exclude this food.
func foodMatchTerms(f FoodItem) []string {
	terms := foodWords(f.Name)
	for _, word := range foodWords(f.Code) {
		terms = append(terms, word)
	}
	for _, term := range f.Terms {
		terms = append(terms, foodWords(term)...)
	}
	return terms
}

// matchesRestriction reports whether a client's restriction text excludes this
// food by name.
//
// Matching is word-based rather than substring-based on purpose. Substring
// matching would let "egg" exclude "eggplant" and "cod" exclude "cod liver oil",
// quietly removing foods the client never objected to; word matching only removes
// a food when the client actually named it or a word that means it.
func matchesRestriction(f FoodItem, words map[string]bool) bool {
	for _, term := range foodMatchTerms(f) {
		if words[term] {
			return true
		}
	}
	return false
}

// matchesGroup reports whether a client's restriction text excludes this food's
// group, for example "no red meat" or "no fish". The phrase is matched against
// the whole normalised entry rather than its individual words, so a multi-word
// exclusion is honoured exactly as written.
func matchesGroup(f FoodItem, phrases []string) (string, bool) {
	for _, phrase := range phrases {
		normalised := strings.Join(foodWords(phrase), " ")
		if normalised == "" {
			continue
		}
		for _, group := range f.Groups {
			for _, candidate := range groupPhrases[group] {
				normalisedCandidate := strings.Join(foodWords(candidate), " ")
				if normalisedCandidate != "" && strings.Contains(normalised, normalisedCandidate) {
					return group, true
				}
			}
		}
	}
	return "", false
}
