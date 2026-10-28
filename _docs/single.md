# Single Container View

ctop provides a rolling, single container view for following metrics

Panels span the terminal width and resize with the window. Use the up and down
keys to scroll when the panels are taller than the terminal.

CPU, network, and I/O plots use the available chart width, with the newest sample
at the right edge. Resizing preserves retained history; each column represents
one sample.
<p align="center"><img width="80%" src="img/single.gif" alt="ctop"/></p>
