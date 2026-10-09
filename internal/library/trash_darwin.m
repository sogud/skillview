#import <Foundation/Foundation.h>
#include <string.h>
char *skillview_trash(const char *path) {
 @autoreleasepool {
  NSURL *source=[NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
  NSURL *destination=nil;
  NSError *error=nil;
  if (![[NSFileManager defaultManager] trashItemAtURL:source resultingItemURL:&destination error:&error]) return NULL;
  const char *result=[[destination path] fileSystemRepresentation];
  return result ? strdup(result) : NULL;
 }
}
