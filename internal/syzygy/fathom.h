#ifndef FATHOM_H
#define FATHOM_H

#include <stdbool.h>

/* Syzygy WDL values */
#define FATHOM_WDL_LOSS        0
#define FATHOM_WDL_BLESSED_LOSS 1
#define FATHOM_WDL_DRAW        2
#define FATHOM_WDL_CURSED_WIN  3
#define FATHOM_WDL_WIN         4

/* Syzygy DTZ values (can be complex, but we'll focus on WDL first) */

/* Initialize the tablebase prober with a path */
bool fathom_init(const char *path);

/* Probe WDL for a position */
/* 
   Parameters:
   white, black: bitboards for each side
   kings, queens, rooks, bishops, knights, pawns: bitboards for each piece type
   ep: en passant square (or 0)
   castling: castling rights (though Syzygy doesn't use them, Fathom needs to know)
   side: side to move (0 for white, 1 for black)
*/
int fathom_probe_wdl(
    unsigned long long white, unsigned long long black,
    unsigned long long kings, unsigned long long queens, unsigned long long rooks,
    unsigned long long bishops, unsigned long long knights, unsigned long long pawns,
    unsigned int ep, unsigned int castling, unsigned int side
);

/* Probe DTZ for a position */
int fathom_probe_dtz(
    unsigned long long white, unsigned long long black,
    unsigned long long kings, unsigned long long queens, unsigned long long rooks,
    unsigned long long bishops, unsigned long long knights, unsigned long long pawns,
    unsigned int ep, unsigned int castling, unsigned int side
);

#endif /* FATHOM_H */