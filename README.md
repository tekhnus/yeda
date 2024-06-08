## About

**yeda** makes Anki cards from a book of your choice.
It selects the best sentences for you to learn the most frequent words.

## Usage

To make English-Russian Anki cards from a book in English:
```
./scripts/run -anki -src English -dst Russian book.txt >cards.txt
```

To see a debug report including the chosen sentences and the learning curve:
```
./scripts/run -report my-book.txt
```

It is recommended to install Nix and run all the commands within `nix develop`.
