#include <stdio.h>
#include "dpiImpl.h"

void CallbackSubscr(void *context, dpiSubscrMessage *message);

void CallbackSubscrDebug(void *context, dpiSubscrMessage *message) {
	fprintf(stderr, "callback called\n");
	CallbackSubscr(context, message);
}
