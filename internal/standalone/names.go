//go:build linux

package standalone

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/containerd/containerd/v2/core/containers"
)

var (
	nameAdjectives = []string{
		"admiring", "adoring", "agitated", "amazing", "angry", "awesome", "beautiful", "blissful", "bold",
		"boring", "brave", "busy", "charming", "clever", "cool", "compassionate", "competent", "confident",
		"crazy", "dazzling", "determined", "distracted", "dreamy", "eager", "ecstatic", "elastic", "elated",
		"elegant", "eloquent", "epic", "exciting", "fervent", "festive", "flamboyant", "focused", "friendly",
		"frosty", "funny", "gallant", "gifted", "goofy", "gracious", "great", "happy", "hardcore", "heuristic",
		"hopeful", "hungry", "infallible", "inspiring", "intelligent", "interesting", "jolly", "jovial",
		"keen", "kind", "laughing", "loving", "lucid", "magical", "modest", "musing", "mystifying", "naughty",
		"nervous", "nice", "nifty", "nostalgic", "objective", "optimistic", "peaceful", "pedantic", "pensive",
		"practical", "precious", "quirky", "quizzical", "recursing", "relaxed", "reverent", "romantic",
		"sad", "serene", "sharp", "silly", "sleepy", "stoic", "strange", "stupefied", "suspicious", "sweet",
		"tender", "thirsty", "trusting", "unruffled", "upbeat", "vibrant", "vigilant", "vigorous",
		"wizardly", "wonderful", "xenodochial", "youthful", "zealous", "zen",
	}
	nameSurnames = []string{
		"albattani", "allen", "almeida", "antonelli", "archimedes", "ardinghelli", "aryabhata", "austin",
		"babbage", "banach", "banzai", "bardeen", "bartik", "bassi", "beaver", "bell", "benz", "bhabha",
		"bhaskara", "black", "blackburn", "blackwell", "bohr", "booth", "borg", "bose", "bouman", "boyd",
		"brahmagupta", "brattain", "brown", "buck", "burnell", "cannon", "carson", "cartwright", "carver",
		"cerf", "chandrasekhar", "chaplygin", "chatelet", "chatterjee", "chaum", "chebyshev", "clarke",
		"cohen", "colden", "cori", "cray", "curie", "curran", "darwin", "davinci", "dewdney", "dhawan",
		"diffie", "dijkstra", "dirac", "driscoll", "dubinsky", "easley", "edison", "einstein", "elbakyan",
		"elgamal", "elion", "ellis", "engelbart", "euclid", "euler", "faraday", "feistel", "fermat",
		"fermi", "feynman", "franklin", "gagarin", "galileo", "galois", "ganguly", "gates", "gauss",
		"germain", "goldberg", "goldstine", "goldwasser", "golick", "goodall", "gould", "greider",
		"grothendieck", "haibt", "hamilton", "haslett", "hawking", "heisenberg", "hermann", "herschel",
		"hertz", "heyrovsky", "hodgkin", "hofstadter", "hoover", "hopper", "hugle", "hypatia", "ishizaka",
		"jackson", "jang", "jemison", "jennings", "jepsen", "johnson", "joliot", "jones", "kalam", "kapitsa",
		"kare", "keldysh", "keller", "kepler", "khayyam", "khorana", "kilby", "kirch", "knuth", "kowalevski",
		"lalande", "lamarr", "lamport", "leakey", "leavitt", "lederberg", "lehmann", "lewin", "lichterman",
		"liskov", "lovelace", "lumiere", "mahavira", "margulis", "matsumoto", "maxwell", "mayer", "mccarthy",
		"mcclintock", "mclaren", "mclean", "mcnulty", "meitner", "mendel", "mendeleev", "merkle", "mestorf",
		"mirzakhani", "montalcini", "moore", "morse", "moser", "murdock", "napier", "nash", "neumann",
		"newton", "nightingale", "nobel", "noether", "northcutt", "noyce", "panini", "pare", "pascal",
		"pasteur", "payne", "perlman", "pike", "poincare", "poitras", "proskuriakova", "ptolemy", "raman",
		"ramanujan", "rhodes", "ride", "riemann", "ritchie", "robinson", "roentgen", "rosalind", "rubin",
		"saha", "sammet", "sanderson", "satoshi", "shamir", "shannon", "shaw", "shirley", "shockley",
		"shtern", "sinoussi", "snyder", "solomon", "spence", "stonebraker", "sutherland", "swanson",
		"swartz", "swirles", "taussig", "tesla", "tharp", "thompson", "torvalds", "tu", "turing", "varahamihira",
		"vaughan", "villani", "visvesvaraya", "volhard", "wescoff", "wilbur", "wiles", "williams", "williamson",
		"wilson", "wing", "wozniak", "wright", "wu", "yalow", "yonath", "zhukovsky",
	}
)

// randomName returns a Docker-style random container name that is not in use.
func randomName(ctx context.Context, store containers.Store) (string, error) {
	for range 10 {
		name := pick(nameAdjectives) + "_" + pick(nameSurnames)
		if _, _, err := loadContainer(ctx, store, name); err != nil {
			return name, nil
		}
	}
	return "", errors.New("failed to generate a unique container name")
}

func pick(from []string) string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(from))))
	if err != nil {
		return from[0]
	}
	return from[n.Int64()]
}
