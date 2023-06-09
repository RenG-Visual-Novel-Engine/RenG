#include <SDL.h>
#include <stdio.h>

Uint32* YUV420ToRGBA8888(Uint8* YPlane, Uint8* UPlane, Uint8* VPlane, int Width, int Height) {
    Uint32* Pixels = (Uint32*)malloc(Width*Height*4+1);

    for (int y = 0; y < Height; y++) {
        for (int x = 0; x < Width; x++) {
            Uint8 ySample = YPlane[y * Width + x];
            Uint8 uSample = UPlane[(y / 2) * (Width / 2) + (x / 2)];
            Uint8 vSample = VPlane[(y / 2) * (Width / 2) + (x / 2)];

            int r = (int)(ySample + 1.402 * (vSample - 128));
            int g = (int)(ySample - 0.344136 * (uSample - 128) - 0.714136 * (vSample - 128));
            int b = (int)(ySample + 1.772 * (uSample - 128));
            int a = 255;
    
            r = (r < 0) ? 0 : ((r > 255) ? 255 : r);
            g = (g < 0) ? 0 : ((g > 255) ? 255 : g);
            b = (b < 0) ? 0 : ((b > 255) ? 255 : b);

            Uint32 rgba = (r << 24) | (g << 16) | (b << 8) | a;

            Pixels[y*Width + x] = rgba;
        }
    }

    return Pixels;
}