#include <SDL.h>

#include <math.h>

void linblur32_core(SDL_Surface* src, SDL_Surface* dst, int radius, int vertical) {

    int c, r;

    int rows, cols;
    int incr, skip;

    unsigned char* srcpixels;
    unsigned char* dstpixels;

    unsigned char* dstp;

    srcpixels = (unsigned char*)src->pixels;
    dstpixels = (unsigned char*)dst->pixels;

    if (vertical) {
        rows = dst->w;
        skip = 4;
        incr = dst->pitch - 4;
        cols = dst->h;
    }
    else {
        rows = dst->h;
        skip = dst->pitch;
        incr = 0;
        cols = dst->w;
    }

    int divisor = radius * 2 + 1;

    for (r = 0; r < rows; r++) {
        // The values of the pixels on the left and right ends of the
        // line.
        unsigned char lr, lg, lb, la;
        unsigned char rr, rg, rb, ra;

        unsigned char* leader = srcpixels + r * skip;
        unsigned char* trailer = leader;
        dstp = dstpixels + r * skip;

        lr = *leader;
        lg = *(leader + 1);
        lb = *(leader + 2);
        la = *(leader + 3);

        int sumr = lr * radius;
        int sumg = lg * radius;
        int sumb = lb * radius;
        int suma = la * radius;

        for (c = 0; c < radius; c++) {
            sumr += *leader++;
            sumg += *leader++;
            sumb += *leader++;
            suma += *leader++;
            leader += incr;
        }

        // left side of the kernel is off of the screen.
        for (c = 0; c < radius; c++) {
            sumr += *leader++;
            sumg += *leader++;
            sumb += *leader++;
            suma += *leader++;
            leader += incr;

            *dstp++ = sumr / divisor;
            *dstp++ = sumg / divisor;
            *dstp++ = sumb / divisor;
            *dstp++ = suma / divisor;
            dstp += incr;

            sumr -= lr;
            sumg -= lg;
            sumb -= lb;
            suma -= la;
        }

        int end = cols - radius - 1;

        // The kernel is fully on the screen.
        for (; c < end; c++) {
            sumr += *leader++;
            sumg += *leader++;
            sumb += *leader++;
            suma += *leader++;
            leader += incr;

            *dstp++ = sumr / divisor;
            *dstp++ = sumg / divisor;
            *dstp++ = sumb / divisor;
            *dstp++ = suma / divisor;
            dstp += incr;

            sumr -= *trailer++;
            sumg -= *trailer++;
            sumb -= *trailer++;
            suma -= *trailer++;
            trailer += incr;
        }

        rr = *leader++;
        rg = *leader++;
        rb = *leader++;
        ra = *leader++;

        // The kernel is off the right side of the screen.
        for (; c < cols; c++) {
            sumr += rr;
            sumg += rg;
            sumb += rb;
            suma += ra;

            *dstp++ = sumr / divisor;
            *dstp++ = sumg / divisor;
            *dstp++ = sumb / divisor;
            *dstp++ = suma / divisor;
            dstp += incr;

            sumr -= *trailer++;
            sumg -= *trailer++;
            sumb -= *trailer++;
            suma -= *trailer++;
            trailer += incr;
        }
    }
}

void linblur24_core(SDL_Surface* src, SDL_Surface* dst, int radius, int vertical) {

    int c, r;

    int rows, cols;
    int incr, skip;

    unsigned char* srcpixels;
    unsigned char* dstpixels;

    unsigned char* dstp;

    srcpixels = (unsigned char*)src->pixels;
    dstpixels = (unsigned char*)dst->pixels;

    if (vertical) {
        rows = dst->w;
        skip = 3;
        incr = dst->pitch - 3;
        cols = dst->h;
    }
    else {
        rows = dst->h;
        skip = dst->pitch;
        incr = 0;
        cols = dst->w;
    }

    int divisor = radius * 2 + 1;

    for (r = 0; r < rows; r++) {
        // The values of the pixels on the left and right ends of the
        // line.
        unsigned char lr, lg, lb;
        unsigned char rr, rg, rb;

        unsigned char* leader = srcpixels + r * skip;
        unsigned char* trailer = leader;
        dstp = dstpixels + r * skip;

        lr = *leader;
        lg = *(leader + 1);
        lb = *(leader + 2);

        int sumr = lr * radius;
        int sumg = lg * radius;
        int sumb = lb * radius;

        for (c = 0; c < radius; c++) {
            sumr += *leader++;
            sumg += *leader++;
            sumb += *leader++;
            leader += incr;
        }

        // left side of the kernel is off of the screen.
        for (c = 0; c < radius; c++) {
            sumr += *leader++;
            sumg += *leader++;
            sumb += *leader++;
            leader += incr;

            *dstp++ = sumr / divisor;
            *dstp++ = sumg / divisor;
            *dstp++ = sumb / divisor;
            dstp += incr;

            sumr -= lr;
            sumg -= lg;
            sumb -= lb;
        }

        int end = cols - radius - 1;

        // The kernel is fully on the screen.
        for (; c < end; c++) {
            sumr += *leader++;
            sumg += *leader++;
            sumb += *leader++;
            leader += incr;

            *dstp++ = sumr / divisor;
            *dstp++ = sumg / divisor;
            *dstp++ = sumb / divisor;
            dstp += incr;

            sumr -= *trailer++;
            sumg -= *trailer++;
            sumb -= *trailer++;
            trailer += incr;
        }

        rr = *leader++;
        rg = *leader++;
        rb = *leader++;

        // The kernel is off the right side of the screen.
        for (; c < cols; c++) {
            sumr += rr;
            sumg += rg;
            sumb += rb;

            *dstp++ = sumr / divisor;
            *dstp++ = sumg / divisor;
            *dstp++ = sumb / divisor;
            dstp += incr;

            sumr -= *trailer++;
            sumg -= *trailer++;
            sumb -= *trailer++;
            trailer += incr;
        }
    }
}

void blur_filters(float sigma, int n, int* wl, int* wu, int* m) {
    *wl = (int)floor(sqrt(12 * sigma * sigma / n + 1));
    if (*wl % 2 == 0) (*wl)--;
    *wu = *wl + 2;
    *m = (int)round(
        (12 * sigma * sigma - n * *wl * *wl - 4 * n * *wl - 3 * n)
        / (-4 * *wl - 4)
    );
}

void blur32_core(SDL_Surface* src, SDL_Surface* ws, SDL_Surface* rv, float xrad, float yrad)
{
	int n = 3; // number of passes, no more than six

	int xl, xu, xm;
	int yl, yu, ym;

	blur_filters(xrad, n, &xl, &xu, &xm);

	if (xrad != yrad) {
		blur_filters(yrad, n, &yl, &yu, &ym);
	}
	else {
		yl = xl; yu = xu; ym = xm;
	}

	for (int i = 0; i < n; i++) {
		int xr = i < xm ? xl : xu;
		linblur32_core(src, ws, xr, 0);
		int yr = i < ym ? yl : yu;
		linblur32_core(ws, rv, yr, 1);
		src = rv;
	}
}

void blur24_core(SDL_Surface* src, SDL_Surface* wrk, SDL_Surface* dst, float xrad, float yrad) {

    int n = 3; // number of passes, no more than six

    int xl, xu, xm;
    int yl, yu, ym;

    blur_filters(xrad, n, &xl, &xu, &xm);

    if (xrad != yrad) {
        blur_filters(yrad, n, &yl, &yu, &ym);
    }
    else {
        yl = xl; yu = xu; ym = xm;
    }

    for (int i = 0; i < n; i++) {
        int xr = i < xm ? xl : xu;
        linblur24_core(src, wrk, xr, 0);
        int yr = i < ym ? yl : yu;
        linblur24_core(wrk, dst, yr, 1);
        src = dst;
    }
}

SDL_Surface* Blur(SDL_Surface* src, float xrad, float yrad) {
    if (src->format->BitsPerPixel == 32) {
        SDL_Surface* ws = SDL_CreateRGBSurfaceWithFormat(0, src->w, src->h, src->format->BitsPerPixel, src->format->format);
	    SDL_Surface* rv = SDL_CreateRGBSurfaceWithFormat(0, src->w, src->h, src->format->BitsPerPixel, src->format->format);
        
        blur32_core(src, ws, rv, xrad, yrad);
        SDL_FreeSurface(rv);
        return ws;
    } else {
        SDL_Surface* ws = SDL_CreateRGBSurfaceWithFormat(0, src->w, src->h, src->format->BitsPerPixel, src->format->format);
	    SDL_Surface* rv = SDL_CreateRGBSurfaceWithFormat(0, src->w, src->h, src->format->BitsPerPixel, src->format->format);
        
        blur24_core(src, ws, rv, xrad, yrad);
        SDL_FreeSurface(rv);
        return ws;
    }
}