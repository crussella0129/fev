package ui

import "math/rand/v2"

// Verb categories. Each tool name routes to one of these via VerbForTool.
// Add more freely — RandomVerb picks uniformly, so longer lists just mean
// less repetition. Style: lowercase first letter, ellipsis at the end,
// active voice, gerund or short participle.
var (
	ThinkingVerbs = []string{
		"Pondering...",
		"Reasoning through this...",
		"Thinking it over...",
		"Considering the options...",
		"Working through it...",
		"Mulling this over...",
		"Connecting the dots...",
		"Following the thread...",
		"Marinating in thought...",
		"Consulting the inner committee...",
		"Untangling the yarn...",
		"Brewing a take...",
		"Squinting thoughtfully...",
	}

	ReadingVerbs = []string{
		"Reading the source...",
		"Scanning the codebase...",
		"Rummaging through files...",
		"Studying the code...",
		"Examining the contents...",
		"Poring over the source...",
		"Decoding ancient runes...",
		"Befriending the code...",
		"Eavesdropping on functions...",
		"Catching up on lore...",
		"Inspecting the artifact...",
	}

	SearchingVerbs = []string{
		"Hunting for matches...",
		"Sifting through the project...",
		"Tracing references...",
		"Searching far and wide...",
		"Following the trail...",
		"Narrowing it down...",
		"Tickling the index...",
		"Bushwhacking through directories...",
		"Sniffing for clues...",
		"Beating the bushes...",
		"Tugging on threads...",
	}

	ExecutingVerbs = []string{
		"Running the command...",
		"Executing...",
		"Waiting on the shell...",
		"Processing...",
		"Working on it...",
		"Poking the shell...",
		"Whispering to the OS...",
		"Cajoling the subprocess...",
		"Wrangling exit codes...",
		"Coaxing the daemon...",
	}

	WritingVerbs = []string{
		"Crafting the code...",
		"Writing changes...",
		"Editing the source...",
		"Applying modifications...",
		"Whittling the bytes...",
		"Conjuring code...",
		"Calligraphing the diff...",
		"Threading the needle...",
		"Persuading the linter...",
	}

	CompactingVerbs = []string{
		"Tidying up context...",
		"Consolidating memory...",
		"Making room to think...",
		"Organizing thoughts...",
		"Tetris-ing the context...",
		"Folding the messages...",
		"Decluttering thoughts...",
		"Vacuuming the buffer...",
	}

	SubagentVerbs = []string{
		"Dispatching helpers...",
		"Agents at work...",
		"Gathering intel...",
		"Coordinating...",
		"Summoning the squad...",
		"Releasing the hounds...",
		"Sending out scouts...",
		"Hailing the away team...",
	}
)

// RandomVerb returns a uniformly random entry from a verb slice. Safe on
// empty input — falls back to a generic "Working..." so a missing category
// never panics or shows an empty spinner.
func RandomVerb(verbs []string) string {
	if len(verbs) == 0 {
		return "Working..."
	}
	return verbs[rand.IntN(len(verbs))]
}

// VerbForTool routes a tool name to the right verb category. Unknown tools
// fall back to the generic ThinkingVerbs — that's the same pool used while
// the LLM is generating between tool calls.
func VerbForTool(toolName string) string {
	switch toolName {
	case "read_file":
		return RandomVerb(ReadingVerbs)
	case "grep", "glob", "list_dir":
		return RandomVerb(SearchingVerbs)
	case "bash", "git":
		return RandomVerb(ExecutingVerbs)
	case "write_file", "edit_file":
		return RandomVerb(WritingVerbs)
	default:
		return RandomVerb(ThinkingVerbs)
	}
}
