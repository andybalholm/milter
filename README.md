# milter
--
    import "milter"

The milter package is a framework for writing milters (mail filters) for
Sendmail and Postfix.

To implement a milter, make a type that implements the Milter interface, listen
on a Unix or TCP socket, and call Serve with that socket and a factory function
that returns instances of your Milter type.
