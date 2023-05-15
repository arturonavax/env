# How I use Delve

Utilizo [Delve](https://github.com/go-delve/delve) como herramienta de
depuracion al desarrollar en [Golang](https://go.dev/).

Instalacion:

```sh
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Iniciar sesion de depuracion

```sh
dlv debug
```

## Iniciar sesion de depuracion en un proceso en ejecusion

```sh
dlv attach <PID>
```

## Iniciar sesion de depuracion en archivos de testing

```sh
dlv test
```

## Saber donde te encuentras

```sh
list
# alias: ls
```

## Ver un statement especifico

```sh
list main.main

# or
list ./main.go:10
```

## Crear un breakpoint

```sh
break main.main
# alias: b

# or specific line
break ./main.go:20

# or relative line based on a statement
break main.main:2

# or when you are already in a statement you can specify the line in the file
break 13
```

Solo puedes crear breakpoints en lineas del statement.

No puedes crear un breakpoint si:

- Es a una linea vacia
- Solo es una linea de cierre de estructura, por ejemplo `}`
- Es una linea de una funcion/metodo/statement que no forma parte de la ruta de
  ejecusion principal. (No puede llegarse a esa linea)

## Saber los breakpoints existentes

```sh
breakpoints
# alias: bp
```

## Limpiar un breakpoint

```sh
clear 1
```

## Limpiar todos los breakpoints

```sh
clearall
```

## Avanzar al siguiente breakpoint

```sh
continue
# alias: c
```

## Avanzar una linea en el breakpoint actual

```sh
next
# alias: n
```

## Ingresar a la ejecusion de una funcion en la linea actual

```sh
step
# alias: s
```

Puede usarse como `next`

## Salir de la ejecusion de una funcion

```sh
stepout
# alias: so
```

## Reiniciar la sesion de depuracion sin perder los breakpoints creados

```sh
restart
# alias: r
```

## Imprimir un valor

```sh
print
# alias: p
```

## Ver todas las variables locales

```sh
locals
```

## Ver y buscar todas las funciones disponibles

```sh
funcs
```

## Examina el valor raw de una direccion de memoria

```sh
examinemem 0xc00007de28
# alias: x
```

## Establecer el valor a una variable

```sh
set x = 11
```

## Imprimir el tipo de datos de una variable

```sh
whatis x
```

## Ver el Assembly de la linea actual

```sh
disassemble
# alias: disass
```

## Operaciones entre Goroutines y Threads

### Ver las goroutines actuales

```sh
goroutines
# alias: grs
```

### Cambiarse a una goroutine en especifico

```sh
goroutine 6
# alias: gr
```

### Ver los threads actuales

```sh
threads
```

### Cambiarse a un thread en especifico

```sh
thread 6
# alias: tr
```

## Preguntas frecuentes sobre Delve (FAQ)

### Como Delve puede ver el codigo fuente del programa al que hace debug?

Delve no puede ver el codigo fuente del programa ya que es un binario, lo que
hace es mostrar el contenido de los archivos ubicados en la ruta donde
fue compilado dicho programa.

Esto puede hacerlo porque internamente el binario contiene como metadata la
ruta original de compilacion, esto es demostrable al realizar una busqueda
con `grep` de la ruta absoluta dentro del binario.

Tambien se puede comprobar esto al modificar una parte del codigo fuente que
se esta visualizando con Delve, podra ver el cambio en tiempo real en Delve,
pero Delve realmente no reaccionara a estos cambios, por ejemplo... al crear
una linea vacia en el codigo fuente que antes si era una linea del statement,
veras que Delve si se puede posicionar en esa "linea vacia", porque en antes
era una linea del statement valido para el.
