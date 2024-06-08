## About

**yeda** makes Anki cards from a book of your choice.
It selects the best sentences for you to learn the most frequent words.

## Usage

To select 25 sentences from the book and create English-to-Russian Anki cards from them:
```
./scripts/run -anki -src English -dst Russian -n 25 some-book.txt >cards.txt
```

To see the debug report showing the selected sentences and the learning curve:
```
./scripts/run -report some-book.txt
```

It is recommended to install Nix and run all the commands within `nix develop`.
