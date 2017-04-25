package io.test.dropwizard;

import java.io.Closeable;
import java.io.IOException;
import java.io.PrintWriter;

import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;


@SuppressWarnings("serial")
public class TestServlet extends HttpServlet {
    @SuppressWarnings("unchecked")
    @Override
    protected void doGet(HttpServletRequest req, HttpServletResponse resp) throws ServletException, IOException {
        PrintWriter out = null;
        try {
            out = new PrintWriter(resp.getWriter());
            out.println("hello world");
            out.flush();
        } finally {
            close(out);
        }
    }

    private static void close(Closeable c) {
        if (c == null) {
            return;
        }

        try {
            c.close();
        } catch (IOException ignore) {
            /* ignore */
        }
    }
}
