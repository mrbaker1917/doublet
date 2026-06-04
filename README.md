# Doublet (Word Ladder) in Go

A CLI word game where you transform one word into another by changing one letter at a time.
Each intermediate step must also be a valid dictionary word.

## Rules

- Start and target must be the same length.
- Each move must change exactly one letter.
- Every move must produce a valid word.
- You must stay within the allowed max changes.

## Run

```bash
go run .
```

You can provide flags instead of prompts:

```bash
go run . -start cat -end dog -difficulty hard -solve
```

## Dictionary Options

Use a built-in preset:

- `-lexicon small` uses `words.txt`.
- `-lexicon large` uses `words-large.txt` (default).

Use your own dictionary file:

```bash
go run . -dict mywords.txt -start cold -end warm -difficulty medium -solve
```

When `-dict` is provided, it overrides `-lexicon`.

## Difficulty Options

- `easy`: max changes = shortest path + 2
- `medium`: max changes = shortest path + 1
- `hard`: max changes = shortest path
- `custom`: requires `-max`

Examples:

```bash
go run . -lexicon small -start cat -end dog -difficulty easy -solve
go run . -lexicon small -start cat -end dog -difficulty custom -max 3 -solve
```

## Interactive Commands

During gameplay:

- `/hint` shows the next step on the shortest path.
- `/solve` reveals the full shortest path.
- `/quit` exits the game.
