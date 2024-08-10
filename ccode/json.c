#include "cJSON.h"

cJSON *parse_json(const char *value)
{
	return cJSON_Parse(value);
}

cJSON *parse_json_with_length(const char *value, size_t buffer_length)
{
	return cJSON_ParseWithLength(value, buffer_length);
}