<img width="6016" height="1504" alt="head and logo flamtap" src="https://github.com/user-attachments/assets/1eeba479-c06e-4e5e-9333-6eaa60c51884" />

A lightweight Go converter with a Gio-based GUI that transforms General MIDI (GM) drum tracks on MIDI channel 10 into four-lane Pro Drums charts for Clone Hero.

Platforms: Windows and Linux.

Features:

- GM drum MIDI conversion
- Four-lane Pro Drums chart generation
- Simple graphical interface built with Gio
- Lightweight implementation in Go

Build & Usage:

Build the project using build.bat on Windows or ./build.sh on Linux. Alternatively, build it manually with Go.
Launch f1amtap.exe on Windows or f1amtap on Linux. In the GUI, select the input MIDI file and specify the output directory.
The generated output folder will be named after the input MIDI file.
The converter generates a notes.chart file without a song.ini. The resulting chart is a starting point and may require manual adjustments.
Review the timing and note grid in Moonscraper Chart Editor and refine the chart to make it playable.
