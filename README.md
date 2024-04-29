Usage:

```
# 1. You'd better install Nix.
nix develop

# 2. Put your book into book.txt.

# 3. Use:

#   To print the learning curve:
./scripts/run -report book.txt

#   To make an html page with cards:
./scripts/run -html book.txt >cards.html

#   To make English-Russian Anki cards from an book in English:
# Put your OpenAI API key in ~/.config/yeda/openai-api-key.txt
./scripts/run -anki -src English -dst Russian book.txt >cards.txt
```
