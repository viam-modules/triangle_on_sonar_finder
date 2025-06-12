# triangle_on_sonar_finder

# Triangle Finder Module

This module provides a vision service that can detect triangles on a sonar screen.

## Configuration

Here's an example configuration:
```json
edit this and change the path:
{
  "camera_name": "camera-1",
  "path_to_templates_directory": "/path/to/templates",
  "threshold": 0.65
}
```

# Compiling

You will need OpenCV 4.11.0 installed to build this! This is available via `brew` on Mac and Linux,
but not through `apt-get`.
