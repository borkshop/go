// +build js

package main

import (
	"borkshop/text"
	"borkshop/webgl"
)

// glCellRenderer is a webgl program for rendering a square grid of cells.
type glCellRenderer struct {
	webgl.Program

	PMatrix webgl.Uniform `glName:"uPMatrix"`
	VP      webgl.Uniform `glName:"uVP"`
	Radius  webgl.Uniform `glName:"uRadius"`

	Vert  webgl.Attrib `glName:"aVert"`
	Color webgl.Attrib `glName:"aColor"`

	Cells webgl.Buffer
}

func (ren *glCellRenderer) Init(gl *webgl.Canvas) (err error) {
	if ren.Program, err = gl.Build(
		webgl.VertexShader(text.Dedent(`
		// Vertex shader for a square grid of cells.

		uniform mat4 uPMatrix;
		uniform vec2 uVP;
		uniform float uRadius;

		attribute vec2 aVert;  // cell center point: x, y
		attribute vec4 aColor; // cell color: rgba

		const vec2 scale = vec2(1.5, sqrt(3.0));

		varying lowp vec4 vColor;

		void main(void) {
			gl_PointSize = uVP.y * abs(uPMatrix[1][1]) * uRadius;
			gl_Position = uPMatrix * vec4(aVert.x, aVert.y, 0.0, 1.0);
			vColor = aColor;
		}
		`)),
		webgl.FragmentShader(text.Dedent(`
		// Trivial paas-thru fragment shader.

		varying lowp vec4 vColor;

		void main(void) {
			gl_FragColor = vColor;
		}
		`)),
	); err != nil {
		return err
	}

	if err := ren.Program.Bind(ren); err != nil {
		return err
	}

	// this.gl.clearColor(0.0, 0.0, 0.0, 0.0);
	// this.gl.uniform1f(this.hexShader.uniform.uRadius, 1);

	return nil
}

// this.gl.enableVertexAttribArray(this.attrs[i]);

// function drawTiles() {
//     this.gl.disableVertexAttribArray(this.hexShader.attr.ang);
//     for (var i = 0; i < this.tileBufferer.tileBuffers.length; ++i) {
//         var tileBuffer = this.tileBufferer.tileBuffers[i];
//         if (!tileBuffer.tiles.length) {
//             continue;
//         }
//         this.gl.bindBuffer(this.gl.ARRAY_BUFFER, tileBuffer.verts.buf);
//         this.gl.vertexAttribPointer(this.hexShader.attr.vert, tileBuffer.verts.width, this.gl.FLOAT, false, 0, 0);
//         this.gl.bindBuffer(this.gl.ARRAY_BUFFER, tileBuffer.colors.buf);
//         this.gl.vertexAttribPointer(this.hexShader.attr.color, tileBuffer.colors.width, this.gl.UNSIGNED_BYTE, true, 0, 0);
//         this.gl.bindBuffer(this.gl.ELEMENT_ARRAY_BUFFER, tileBuffer.elements.buf);
//         this.gl.drawElements(this.gl.POINTS, tileBuffer.usedElements, this.gl.UNSIGNED_SHORT, 0);
//     }
// };

// this.gl.disableVertexAttribArray(this.attrs[i]);
// this.gl.finish();

// fixAspectRatio(
//     this.gl.drawingBufferWidth / this.gl.drawingBufferHeight,
//     this.topLeft, this.bottomRight);
// mat4.ortho(this.perspectiveMatrix,
//     this.topLeft.x, this.bottomRight.x,
//     this.bottomRight.y, this.topLeft.y,
//     -1, 1);
// this.gl.uniformMatrix4fv(this.hexShader.uniform.uPMatrix, false, this.perspectiveMatrix);

// function fixAspectRatio(aspectRatio, topLeft, bottomRight) {
//     var gridWidth = bottomRight.x - topLeft.x;
//     var gridHeight = bottomRight.y - topLeft.y;
//     var ratio = gridWidth / gridHeight;
//     if (ratio < aspectRatio) {
//         var dx = gridHeight * aspectRatio / 2 - gridWidth / 2;
//         topLeft.x -= dx;
//         bottomRight.x += dx;
//     } else if (ratio > aspectRatio) {
//         var dy = gridWidth / aspectRatio / 2 - gridHeight / 2;
//         topLeft.y -= dy;
//         bottomRight.y += dy;
//     }
// }

// this.gl.viewport(0, 0, width, height);
// this.gl.uniform2f(this.hexShader.uniform.uVP, this.canvas.width, this.canvas.height);

// this.gl.vertexAttribPointer(this.hexShader.attr.color, tileBuffer.colors.width, this.gl.UNSIGNED_BYTE, true, 0, 0);
// this.gl.vertexAttribPointer(this.hexShader.attr.vert, tileBuffer.verts.width, this.gl.FLOAT, false, 0, 0);

// this.gl.bindBuffer(this.gl.ARRAY_BUFFER, colorBuf);
// this.gl.vertexAttribPointer(hexShader.attr.color, 1, this.gl.UNSIGNED_BYTE, true, 0, 0);

// this.gl.bindBuffer(this.gl.ARRAY_BUFFER, tileBuffer.verts.buf);
// this.gl.vertexAttribPointer(this.hexShader.attr.vert, tileBuffer.verts.width, this.gl.FLOAT, false, 0, 0);

// this.gl.bindBuffer(this.gl.ARRAY_BUFFER, tileBuffer.colors.buf);
// this.gl.vertexAttribPointer(this.hexShader.attr.color, tileBuffer.colors.width, this.gl.UNSIGNED_BYTE, true, 0, 0);

// this.gl.uniform1f(this.hexShader.uniform.uRadius, 1);
// this.gl.uniform2f(this.hexShader.uniform.uVP, this.canvas.width, this.canvas.height);
// this.gl.uniformMatrix4fv(this.hexShader.uniform.uPMatrix, false, this.perspectiveMatrix);

// this.gl.drawElements(this.gl.POINTS, tileBuffer.usedElements, this.gl.UNSIGNED_SHORT, 0);

// this.verts = new Float32Array(this.cap * 4);
// this.colors = new Uint8Array(this.cap * 1);

// this.gl.drawArrays(this.gl.POINTS, 0, this.len);

// this.gl.bindBuffer(this.gl.ELEMENT_ARRAY_BUFFER, tileBuffer.elements.buf);
// this.gl.drawElements(this.gl.POINTS, tileBuffer.usedElements, this.gl.UNSIGNED_SHORT, 0);
