package nutrition_assignment

import "sort"

// buildCatalog returns the curated catalog.
//
// The set is chosen for coverage rather than exhaustiveness: every diet the
// questionnaire accepts, and every allergen token, must leave enough eligible
// foods to fill a full day at the highest supported meal count. A client asking
// for eight meals plus six snacks with a restrictive combination of allergies
// must still be servable, otherwise generation would fail for a legitimate intake.
//
// Energy and macro values are per standard serving and rounded to whole units.
// They are planning figures for a general template, not a clinical reference
// table, and the generator's job is to compose them into a day that hits the
// energy and protein targets derived from the intake.
//
// Halal and kosher flags describe ingredient composition only: no pork, no
// alcohol, no gelatin, and for dairy and meat the client must still verify
// certification and rennet sourcing. The generator states this caveat explicitly
// on the plan rather than implying a certification it cannot verify.
func buildCatalog() []FoodItem {
	items := []FoodItem{
		// Proteins.
		{
			Code: "chicken_breast", Name: "Chicken breast", Category: FoodCategoryProtein,
			Serving: 120, Unit: "g", Calories: 198, Protein: 37, Carbs: 0, Fat: 4.3, Fiber: 0,
			Omnivore: true, Flexitarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupPoultry},
		},
		{
			Code: "turkey_breast", Name: "Turkey breast", Category: FoodCategoryProtein,
			Serving: 120, Unit: "g", Calories: 157, Protein: 35, Carbs: 0, Fat: 2, Fiber: 0,
			Omnivore: true, Flexitarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupPoultry},
		},
		{
			Code: "lean_beef", Name: "Lean beef mince", Category: FoodCategoryProtein,
			Serving: 100, Unit: "g", Calories: 176, Protein: 26, Carbs: 0, Fat: 9, Fiber: 0,
			Omnivore: true, Flexitarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupRedMeat}, Terms: []string{"beef", "mince"},
		},
		{
			Code: "salmon_fillet", Name: "Salmon fillet", Category: FoodCategoryProtein,
			Serving: 120, Unit: "g", Calories: 250, Protein: 26, Carbs: 0, Fat: 16, Fiber: 0,
			Omnivore: true, Flexitarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenFish}, Groups: []string{FoodGroupFish},
		},
		{
			Code: "cod_fillet", Name: "Cod fillet", Category: FoodCategoryProtein,
			Serving: 120, Unit: "g", Calories: 118, Protein: 26, Carbs: 0, Fat: 0.8, Fiber: 0,
			Omnivore: true, Flexitarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenFish}, Groups: []string{FoodGroupFish},
		},
		{
			Code: "tuna_canned", Name: "Tuna in water", Category: FoodCategoryProtein,
			Serving: 120, Unit: "g", Calories: 116, Protein: 26, Carbs: 0, Fat: 1, Fiber: 0,
			Omnivore: true, Flexitarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenFish}, Groups: []string{FoodGroupFish},
		},
		{
			Code: "whole_egg", Name: "Whole egg", Category: FoodCategoryProtein,
			Serving: 50, Unit: "g", Calories: 72, Protein: 6.3, Carbs: 0.4, Fat: 5, Fiber: 0,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenEgg}, Groups: []string{FoodGroupEgg},
		},
		{
			Code: "egg_white", Name: "Egg white", Category: FoodCategoryProtein,
			Serving: 60, Unit: "g", Calories: 34, Protein: 7.2, Carbs: 0.5, Fat: 0.1, Fiber: 0,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenEgg}, Groups: []string{FoodGroupEgg},
		},
		{
			Code: "firm_tofu", Name: "Firm tofu", Category: FoodCategoryProtein,
			Serving: 150, Unit: "g", Calories: 144, Protein: 17, Carbs: 3, Fat: 9, Fiber: 2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenSoy},
		},
		{
			Code: "tempeh", Name: "Tempeh", Category: FoodCategoryProtein,
			Serving: 150, Unit: "g", Calories: 219, Protein: 22, Carbs: 9, Fat: 12, Fiber: 8,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenSoy},
		},
		{
			Code: "seitan", Name: "Seitan", Category: FoodCategoryProtein,
			Serving: 120, Unit: "g", Calories: 143, Protein: 24, Carbs: 14, Fat: 1.7, Fiber: 0.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "greek_yogurt", Name: "Greek yogurt", Category: FoodCategoryProtein,
			Serving: 170, Unit: "g", Calories: 100, Protein: 17, Carbs: 6, Fat: 0, Fiber: 0,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Pescatarian: true,
			Allergens: []string{AllergenMilk}, Groups: []string{FoodGroupDairy},
			Terms: []string{"yoghurt"},
		},
		{
			Code: "cottage_cheese", Name: "Cottage cheese", Category: FoodCategoryProtein,
			Serving: 150, Unit: "g", Calories: 145, Protein: 25, Carbs: 5, Fat: 3, Fiber: 0,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Pescatarian: true,
			Allergens: []string{AllergenMilk}, Groups: []string{FoodGroupDairy},
		},
		{
			Code: "whey_protein", Name: "Whey protein", Category: FoodCategoryProtein,
			Serving: 30, Unit: "g", Calories: 120, Protein: 24, Carbs: 3, Fat: 2, Fiber: 1,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Pescatarian: true,
			Allergens: []string{AllergenMilk}, Groups: []string{FoodGroupDairy},
		},
		{
			Code: "pea_protein", Name: "Pea protein", Category: FoodCategoryProtein,
			Serving: 30, Unit: "g", Calories: 110, Protein: 22, Carbs: 2, Fat: 2, Fiber: 1,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Groups: []string{FoodGroupLegumes},
		},
		{
			Code: "lentils_cooked", Name: "Cooked lentils", Category: FoodCategoryProtein,
			Serving: 200, Unit: "g", Calories: 232, Protein: 18, Carbs: 40, Fat: 0.8, Fiber: 16,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupLegumes},
		},
		{
			Code: "chickpeas_cooked", Name: "Cooked chickpeas", Category: FoodCategoryProtein,
			Serving: 200, Unit: "g", Calories: 328, Protein: 18, Carbs: 54, Fat: 5, Fiber: 13,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupLegumes},
		},
		{
			Code: "edamame", Name: "Edamame", Category: FoodCategoryProtein,
			Serving: 150, Unit: "g", Calories: 189, Protein: 19, Carbs: 14, Fat: 8, Fiber: 8,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenSoy}, Groups: []string{FoodGroupLegumes},
		},

		// Carbohydrates.
		{
			Code: "rolled_oats", Name: "Rolled oats", Category: FoodCategoryCarbohydrate,
			Serving: 50, Unit: "g", Calories: 195, Protein: 6.5, Carbs: 33, Fat: 3.5, Fiber: 5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "white_rice", Name: "White rice", Category: FoodCategoryCarbohydrate,
			Serving: 150, Unit: "g", Calories: 195, Protein: 4.3, Carbs: 43, Fat: 0.4, Fiber: 0.6,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "brown_rice", Name: "Brown rice", Category: FoodCategoryCarbohydrate,
			Serving: 150, Unit: "g", Calories: 165, Protein: 3.8, Carbs: 34, Fat: 1.3, Fiber: 2.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "wholewheat_pasta", Name: "Wholewheat pasta", Category: FoodCategoryCarbohydrate,
			Serving: 180, Unit: "g", Calories: 214, Protein: 9, Carbs: 44, Fat: 1.1, Fiber: 7,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "white_pasta", Name: "Pasta", Category: FoodCategoryCarbohydrate,
			Serving: 180, Unit: "g", Calories: 262, Protein: 9, Carbs: 52, Fat: 1.3, Fiber: 3,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "buckwheat", Name: "Buckwheat", Category: FoodCategoryCarbohydrate,
			Serving: 150, Unit: "g", Calories: 155, Protein: 5.7, Carbs: 33, Fat: 1, Fiber: 3,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "quinoa_cooked", Name: "Cooked quinoa", Category: FoodCategoryCarbohydrate,
			Serving: 150, Unit: "g", Calories: 180, Protein: 6.6, Carbs: 32, Fat: 3, Fiber: 5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
		},
		{
			Code: "couscous", Name: "Couscous", Category: FoodCategoryCarbohydrate,
			Serving: 150, Unit: "g", Calories: 165, Protein: 5, Carbs: 34, Fat: 0.5, Fiber: 2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "sweet_potato", Name: "Sweet potato", Category: FoodCategoryCarbohydrate,
			Serving: 200, Unit: "g", Calories: 180, Protein: 4, Carbs: 41, Fat: 0.3, Fiber: 6.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "potato", Name: "Potato", Category: FoodCategoryCarbohydrate,
			Serving: 200, Unit: "g", Calories: 154, Protein: 4, Carbs: 34, Fat: 0.2, Fiber: 2.2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "wholemeal_bread", Name: "Wholemeal bread", Category: FoodCategoryCarbohydrate,
			Serving: 80, Unit: "g", Calories: 198, Protein: 8, Carbs: 34, Fat: 2.5, Fiber: 5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "sourdough", Name: "Sourdough bread", Category: FoodCategoryCarbohydrate,
			Serving: 70, Unit: "g", Calories: 180, Protein: 7, Carbs: 33, Fat: 1.5, Fiber: 2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "wholewheat_tortilla", Name: "Wholewheat tortilla", Category: FoodCategoryCarbohydrate,
			Serving: 60, Unit: "g", Calories: 160, Protein: 5, Carbs: 27, Fat: 3, Fiber: 4,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenGluten}, Groups: []string{FoodGroupGrains},
		},
		{
			Code: "corn_tortilla", Name: "Corn tortilla", Category: FoodCategoryCarbohydrate,
			Serving: 30, Unit: "g", Calories: 82, Protein: 2, Carbs: 17, Fat: 1, Fiber: 1.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
		},
		{
			Code: "rice_cakes", Name: "Rice cakes", Category: FoodCategoryCarbohydrate,
			Serving: 18, Unit: "g", Calories: 70, Protein: 1.4, Carbs: 15, Fat: 0.5, Fiber: 0.6,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "bagel", Name: "Bagel", Category: FoodCategoryCarbohydrate,
			Serving: 95, Unit: "g", Calories: 257, Protein: 10, Carbs: 50, Fat: 1.5, Fiber: 2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenGluten, AllergenSesame}, Groups: []string{FoodGroupGrains},
		},

		// Vegetables.
		{
			Code: "broccoli", Name: "Broccoli", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 51, Protein: 4.2, Carbs: 10, Fat: 0.6, Fiber: 5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "spinach", Name: "Spinach", Category: FoodCategoryVegetable,
			Serving: 100, Unit: "g", Calories: 23, Protein: 2.9, Carbs: 3.6, Fat: 0.4, Fiber: 2.2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "kale", Name: "Kale", Category: FoodCategoryVegetable,
			Serving: 100, Unit: "g", Calories: 35, Protein: 2.9, Carbs: 4.4, Fat: 1.5, Fiber: 4.1,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "mixed_salad", Name: "Mixed salad leaves", Category: FoodCategoryVegetable,
			Serving: 80, Unit: "g", Calories: 14, Protein: 1.1, Carbs: 2.2, Fat: 0.1, Fiber: 1.4,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "green_beans", Name: "Green beans", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 44, Protein: 2.5, Carbs: 10, Fat: 0.3, Fiber: 4,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "cauliflower", Name: "Cauliflower", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 38, Protein: 2.5, Carbs: 7.5, Fat: 0.4, Fiber: 4.2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "courgette", Name: "Courgette", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 24, Protein: 2.1, Carbs: 4.5, Fat: 0.6, Fiber: 1.8,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "mushrooms", Name: "Mushrooms", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 33, Protein: 4.8, Carbs: 4.5, Fat: 0.5, Fiber: 1.8,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "frozen_peas", Name: "Frozen peas", Category: FoodCategoryVegetable,
			Serving: 100, Unit: "g", Calories: 81, Protein: 5.4, Carbs: 14, Fat: 0.4, Fiber: 5.7,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupLegumes},
		},
		{
			Code: "asparagus", Name: "Asparagus", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 30, Protein: 3, Carbs: 5, Fat: 0.2, Fiber: 3,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "carrots", Name: "Carrots", Category: FoodCategoryVegetable,
			Serving: 100, Unit: "g", Calories: 41, Protein: 0.9, Carbs: 10, Fat: 0.2, Fiber: 2.8,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "bell_pepper", Name: "Bell pepper", Category: FoodCategoryVegetable,
			Serving: 120, Unit: "g", Calories: 37, Protein: 1.2, Carbs: 7, Fat: 0.5, Fiber: 2.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupNightshade},
		},
		{
			Code: "tomatoes", Name: "Tomatoes", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 27, Protein: 1.3, Carbs: 5.6, Fat: 0.3, Fiber: 1.6,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupNightshade},
		},
		{
			Code: "cucumber", Name: "Cucumber", Category: FoodCategoryVegetable,
			Serving: 150, Unit: "g", Calories: 24, Protein: 1, Carbs: 5.5, Fat: 0.3, Fiber: 1.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "onions", Name: "Onions", Category: FoodCategoryVegetable,
			Serving: 100, Unit: "g", Calories: 40, Protein: 1.1, Carbs: 9.3, Fat: 0.1, Fiber: 1.7,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},

		// Fruit.
		{
			Code: "banana", Name: "Banana", Category: FoodCategoryFruit,
			Serving: 120, Unit: "g", Calories: 107, Protein: 1.3, Carbs: 27, Fat: 0.4, Fiber: 3.1,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "apple", Name: "Apple", Category: FoodCategoryFruit,
			Serving: 180, Unit: "g", Calories: 95, Protein: 0.5, Carbs: 25, Fat: 0.3, Fiber: 4.3,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "blueberries", Name: "Blueberries", Category: FoodCategoryFruit,
			Serving: 100, Unit: "g", Calories: 57, Protein: 0.7, Carbs: 14, Fat: 0.3, Fiber: 2.4,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "strawberries", Name: "Strawberries", Category: FoodCategoryFruit,
			Serving: 150, Unit: "g", Calories: 48, Protein: 1, Carbs: 11.4, Fat: 0.5, Fiber: 3,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "orange", Name: "Orange", Category: FoodCategoryFruit,
			Serving: 180, Unit: "g", Calories: 84, Protein: 1.6, Carbs: 20, Fat: 0.4, Fiber: 4.2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "kiwi", Name: "Kiwi", Category: FoodCategoryFruit,
			Serving: 100, Unit: "g", Calories: 61, Protein: 1.1, Carbs: 15, Fat: 0.5, Fiber: 3,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "mango", Name: "Mango", Category: FoodCategoryFruit,
			Serving: 150, Unit: "g", Calories: 99, Protein: 1.6, Carbs: 25, Fat: 0.6, Fiber: 2.6,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},
		{
			Code: "avocado", Name: "Avocado", Category: FoodCategoryFruit,
			Serving: 100, Unit: "g", Calories: 160, Protein: 2, Carbs: 9, Fat: 15, Fiber: 7,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},

		// Fats and add-ons.
		{
			Code: "olive_oil", Name: "Olive oil", Category: FoodCategoryFat,
			Serving: 10, Unit: "ml", Calories: 90, Protein: 0, Carbs: 0, Fat: 10, Fiber: 0,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Terms: []string{"oil"},
		},
		{
			Code: "rapeseed_oil", Name: "Rapeseed oil", Category: FoodCategoryFat,
			Serving: 10, Unit: "ml", Calories: 90, Protein: 0, Carbs: 0, Fat: 10, Fiber: 0,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Terms: []string{"oil"},
		},
		{
			Code: "chia_seeds", Name: "Chia seeds", Category: FoodCategoryFat,
			Serving: 15, Unit: "g", Calories: 73, Protein: 2.5, Carbs: 6, Fat: 4.7, Fiber: 5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Groups: []string{FoodGroupSeeds},
		},
		{
			Code: "flaxseed", Name: "Flaxseed", Category: FoodCategoryFat,
			Serving: 10, Unit: "g", Calories: 53, Protein: 1.8, Carbs: 3, Fat: 4.2, Fiber: 3.9,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Groups: []string{FoodGroupSeeds},
		},
		{
			Code: "pumpkin_seeds", Name: "Pumpkin seeds", Category: FoodCategoryFat,
			Serving: 20, Unit: "g", Calories: 112, Protein: 6, Carbs: 2, Fat: 9, Fiber: 1.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Groups: []string{FoodGroupSeeds},
		},
		{
			Code: "almonds", Name: "Almonds", Category: FoodCategoryFat,
			Serving: 30, Unit: "g", Calories: 174, Protein: 6.4, Carbs: 6, Fat: 15, Fiber: 3.8,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenTreeNut}, Groups: []string{FoodGroupNuts},
		},
		{
			Code: "walnuts", Name: "Walnuts", Category: FoodCategoryFat,
			Serving: 30, Unit: "g", Calories: 196, Protein: 4.6, Carbs: 4, Fat: 19.6, Fiber: 2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenTreeNut}, Groups: []string{FoodGroupNuts},
		},
		{
			Code: "peanut_butter", Name: "Peanut butter", Category: FoodCategoryFat,
			Serving: 20, Unit: "g", Calories: 118, Protein: 5, Carbs: 4, Fat: 10, Fiber: 1.9,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true,
			Allergens: []string{AllergenPeanut}, Groups: []string{FoodGroupNuts},
		},
		{
			Code: "tahini", Name: "Tahini", Category: FoodCategoryFat,
			Serving: 15, Unit: "g", Calories: 89, Protein: 2.6, Carbs: 3.2, Fat: 8, Fiber: 1.4,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
			Allergens: []string{AllergenSesame}, Groups: []string{FoodGroupSeeds},
		},
		{
			Code: "dark_chocolate", Name: "Dark chocolate", Category: FoodCategoryFat,
			Serving: 20, Unit: "g", Calories: 118, Protein: 2, Carbs: 6, Fat: 9, Fiber: 2.2,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Groups: []string{FoodGroupAddedSugar},
		},
		{
			Code: "popcorn", Name: "Popcorn", Category: FoodCategoryFat,
			Serving: 30, Unit: "g", Calories: 125, Protein: 3.6, Carbs: 24, Fat: 1.6, Fiber: 4.5,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true, Halal: true, Kosher: true,
		},

		// Dairy alternatives, so a vegan or dairy-free client still has protein
		// and calcium sources without a separate fortified-only entry.
		{
			Code: "soy_milk", Name: "Soy milk", Category: FoodCategoryProtein,
			Serving: 250, Unit: "ml", Calories: 80, Protein: 7, Carbs: 4, Fat: 4, Fiber: 1,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
			Allergens: []string{AllergenSoy},
		},
		{
			Code: "coconut_yogurt", Name: "Coconut yogurt", Category: FoodCategoryProtein,
			Serving: 150, Unit: "g", Calories: 110, Protein: 1, Carbs: 6, Fat: 9, Fiber: 1,
			Omnivore: true, Flexitarian: true, Vegetarian: true, Vegan: true, Pescatarian: true,
		},
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	return items
}
