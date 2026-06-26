#ifdef __x86_64__
/* Force binding to pre-2.32/2.34 glibc symbol versions.
 * Debian Bookworm merged libpthread into libc (glibc 2.34), which caused
 * pthread symbols to appear at GLIBC_2.34. These symbols exist at older
 * versions too; forcing them lets the binary run on Ubuntu 18.04 (glibc 2.27). */
__asm__(".symver pthread_create,pthread_create@GLIBC_2.1");
__asm__(".symver pthread_key_create,pthread_key_create@GLIBC_2.0");
__asm__(".symver pthread_setspecific,pthread_setspecific@GLIBC_2.0");
__asm__(".symver pthread_getattr_np,pthread_getattr_np@GLIBC_2.2.5");
__asm__(".symver pthread_sigmask,pthread_sigmask@GLIBC_2.0");
__asm__(".symver pthread_detach,pthread_detach@GLIBC_2.0");
__asm__(".symver pthread_attr_getstacksize,pthread_attr_getstacksize@GLIBC_2.1");
__asm__(".symver pthread_attr_getstack,pthread_attr_getstack@GLIBC_2.1");
__asm__(".symver __libc_start_main,__libc_start_main@GLIBC_2.0");
__asm__(".symver res_search,res_search@GLIBC_2.0");
#endif
