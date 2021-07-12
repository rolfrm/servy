#include <stdio.h>
#include <unistd.h>
int main(int argc, char ** argv){
  long long unsigned len[1] = {0};
  int l = read(STDIN_FILENO, len, sizeof(len[0]));
  if(l <= 0) return 1;
  printf("%llu", len[0]);
  return 0;
}
