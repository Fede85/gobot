package nanopineo

var pins = map[string]sysfsPin{
	"GPIOG11": {
		pin:    203,
		pwmPin: -1,
	},
	"GPIOC0": {
		pin:    64,
		pwmPin: -1,
	},
	"GPIOC1": {
		pin:    65,
		pwmPin: -1,
	},
	"GPIOC2": {
		pin:    66,
		pwmPin: -1,
	},
	"GPIOC3": {
		pin:    67,
		pwmPin: -1,
	},
	"GPIOA6": {
		pin:    6,
		pwmPin: -1,
	},
	"GPIOA2": {
		pin:    14,
		pwmPin: -1,
	},
	"GPIOA3": {
		pin:    16,
		pwmPin: -1,
	},
}
