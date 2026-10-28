# Single Container View

ctop provides a rolling, single container view for following metrics

Panels span the terminal width and resize with the window. Use the up and down
keys to scroll when the panels are taller than the terminal.

CPU, memory, network, and I/O plots use the available chart width, with the newest
sample at the right edge. Resizing preserves retained history; each column
represents one sample.

The CPU plot uses a connected Unicode Braille line for finer vertical detail.
Its percentage scale starts at zero, and idle samples share the zero baseline.
A terminal font with Braille support is sufficient; Nerd Fonts are not required.
<p align="center"><img width="80%" src="img/single.gif" alt="ctop"/></p>
