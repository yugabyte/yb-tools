%{
/*
* lexer for YugabyteDB log files
*
* Author: Jake Angerman, July 2022
*/

#include <string.h>
#include <stdlib.h>
#include <regex.h>
#include <time.h>

#define MAX_BUF_SIZE (1024*1024)
#define MAX_INPUT_FILES 256

typedef struct {
    yyscan_t scanner; /* reentrant scanner state */
    FILE *fp; /* input file */
    char *filename; /* input file name */
    char *hostname; /* hostname extracted from preamble */
    char output_buf[MAX_BUF_SIZE]; /* the line written to output */
    char *output; /* current position in the output buffer */
    int line_number; /* current line number for error reporting purposes */
    long long epoch_timestamp; /* timestamp for the current line */
    int regex_match; /* did we find a regex match? */
    int eof; /* reached end of file? */
    int completed; /* done with this file? */
} input_state;

input_state input_states[MAX_INPUT_FILES];
input_state *current_state; /* the currently active input state */
int num_input_files;
int num_files_left_to_process;

char *regex_str = NULL;
regex_t regex;

#define LEXER_SUCCESS 1
#define LEXER_EOF 0

/* I won't use YY_USER_ACTION since this gives me more flexibility */
#define COPY_TO_OUTPUT(yytext, yyleng) \
    strncpy(current_state->output, yytext, yyleng); \
    current_state->output += yyleng;

%}

%x MULTILINE PREAMBLE IGNORE
%option reentrant

%%

^Log\ file\ created\ at:.*/\n {
	BEGIN PREAMBLE;
    }

<PREAMBLE>^Running\ on\ machine:\ .*/\n {
	current_state->hostname = strdup(strchr(yytext, ':')+2);
	BEGIN IGNORE;
    }

<PREAMBLE>\n {
	/* separate rule since \n was put back for easier string extraction */
	current_state->line_number++;
    }

<IGNORE>.*\n {
	/* gobble up and ignore it */
	current_state->line_number++;
}

<IGNORE>.*\n/[IWEF][0-9][0-9][0-9][0-9]\  {
        /* Everything after the forward slash is put back into the input stream. This is called a "trailing context". */
	BEGIN INITIAL;
    }

^[IWEF][0-9][0-9][0-9][0-9]\ [0-9][0-9]:[0-9][0-9]:[0-9][0-9]\.[0-9][0-9][0-9][0-9][0-9][0-9] {
	/* glog example: I0726 18:43:03.516417 */
	/* Note having this rule in INITIAL lets us parse malformed logs */
	BEGIN MULTILINE;

	/* process date */
	struct tm ti = {0};
	int us; /* microseconds */

	if (sscanf(yytext+1, "%2d%2d %d:%d:%d.%d", &ti.tm_mon, &ti.tm_mday, &ti.tm_hour, &ti.tm_min, &ti.tm_sec, &us) != 6) {
		fprintf(stderr, "sscanf() failed\n.");
	}
	ti.tm_year = 2022 - 1900; /* doesn't matter since GLOG format has no year, I only need something relative */
	ti.tm_mon -= 1; /* month of year (0-11) */
	current_state->epoch_timestamp = (mktime(&ti) * 1000000) + us;

	COPY_TO_OUTPUT(yytext, yyleng);

	if (current_state->hostname) {
		/* insert hostname */
		const int bufsize = 256;
		char hostname[bufsize];
		snprintf(hostname, bufsize, " <%s>", current_state->hostname);
		COPY_TO_OUTPUT(hostname, strlen(hostname));
	}
    }

<MULTILINE>.* {
        /* read one or more lines while in multi-line mode */
        COPY_TO_OUTPUT(yytext, yyleng);
    }

<MULTILINE>.*/\n[IWEF][0-9][0-9][0-9][0-9]\  {
        /* Note "trailing context" here with trailing space */
        /* end of multiline */
        BEGIN INITIAL;

	COPY_TO_OUTPUT(yytext, yyleng);
    }

<INITIAL,MULTILINE>\n {
	/* newline */
	/* Note line_number is different than a lexical "log entry" since the latter could span multiple lines */

	current_state->line_number++;
	*current_state->output++ = '\n';
	*current_state->output = '\0';

	/* check for regex match (unless we've found one already) */
	if ((NULL != regex_str) && (!current_state->regex_match) && (0 == regexec(&regex, current_state->output_buf, 0, NULL, 0)) )
	{
		current_state->regex_match = 1;
	}

	return LEXER_SUCCESS;
    }

. {
        /* unrecognized token */
        fprintf(stderr, "unrecognized token %s in %s on line %d\n", yytext, current_state->filename, current_state->line_number);
    }

%%

void read_next_line(input_state * const state)
{
        state->output = &state->output_buf[0];
        state->output_buf[0] = '\0';
        state->regex_match = 0;
	if (LEXER_EOF == yylex(state->scanner)) /* yylex() */
	{
	    state->eof = 1;
	}
}

input_state *find_oldest_input_line(void)
{
        input_state *lowest = current_state;
        int choose_next_available_file = 0;
        int i;

        if (current_state->completed)
        {
            /* need to choose next available file to be the oldest */
            choose_next_available_file = 1;
        }

        for (i=0; i < num_input_files; i++)
        {
            input_state *this_state = &input_states[i];
            if (!this_state->completed)
            {
                if (choose_next_available_file)
                {
                    lowest = this_state;
                    choose_next_available_file = 0;
                }
                else if (this_state->epoch_timestamp < lowest->epoch_timestamp)
                {
                    lowest = this_state;
                }
            }
        }
        return lowest;
}

int main(int argc, char **argv)
{
        const char *program_name = *argv;
        int i;

        argc--; argv++; /* consume $0 */
        if (argc >= 2 && strcmp("-v", *argv) == 0)
        {
            /* regular expression option */
            argv++;
            regex_str = *argv++;
            argc -= 2;
            //printf("DEBUG: regex_str = %s\n", regex_str);

            int result = regcomp(&regex, regex_str, REG_EXTENDED|REG_NOSUB|REG_NEWLINE);
            if (0 != result)
            {
                fprintf(stderr, "fatal: regular expression compilation failed for %s\n", regex_str);
                exit(1);
            }
        }

        if (0 == argc)
        {
            fprintf(stderr, "USAGE: %s [-v regexp] file.log [file2.log ...]\n"
                            "   -v regexp       reverse the search for an extended regular expression, just like egrep -v\n\n"
                            "This program chronologically sorts tserver and master log files into a single stream to stdout.\n"
                            "You can also use a regular expression to filter out noise.\n"
                            "Log files must be in UNIX \\n format and not Windows \\r\\n format.\n"
                            "This program writes to stdout, so you must redirect its output.\n"
                            "Example: %s -v '(ignore1|ignore2|ignore3)' *log > sorted 2> errors\n"
                             , program_name, program_name
                            );
            exit(1);
        }

        num_input_files = argc;
        num_files_left_to_process = num_input_files;

	/* initialize data structures */
        for (i=0; i < num_input_files; i++)
        {
            current_state = &input_states[i];
            current_state->line_number = 1;
            current_state->eof = 0;
            current_state->completed = 0;
            current_state->epoch_timestamp = 0;
            current_state->hostname = NULL;
            current_state->filename = argv[i];
            if ((current_state->fp = fopen(current_state->filename, "r")) == NULL)
            {
                perror(current_state->filename);
                exit(1);
            }

            /* read the very first line from each file */
            yylex_init(&current_state->scanner);
            yyset_in(current_state->fp, current_state->scanner);
            read_next_line(current_state);
        }

	/* main loop */
        while (num_files_left_to_process > 0)
        {
            current_state = find_oldest_input_line();

	    /* print out the buffer */
            if (!current_state->completed && !current_state->regex_match)
            {
                printf("%s", current_state->output_buf);
                //printf("DEBUG: just printed line number = %i\n", current_state->line_number-1);
            }

            if (current_state->eof) {
		current_state->completed = 1;
		num_files_left_to_process--;
		continue;
	    }

            /* read another line to replace the one we just consumed */
            read_next_line(current_state);
        }

	/* shutdown */
        for (i=0; i < num_input_files; i++)
        {
            yylex_destroy(input_states[i].scanner);
            free(input_states[i].hostname);
        }
        exit(0);

} /* main */
