package io.test.dropwizard;

import com.codahale.metrics.Counter;
import com.codahale.metrics.Gauge;
import com.codahale.metrics.Meter;
import com.codahale.metrics.MetricRegistry;
import com.codahale.metrics.Timer;
import com.codahale.metrics.servlets.MetricsServlet;

/**
 *
 * MetricsServletContextListener is a listener class that needs to be added to all assertion
 * web application's web.xml in order to expose the MetricsRegistry which maintains all the
 * metrics that are being tracked.
 *
 */

public class MetricsServletContextListener extends MetricsServlet.ContextListener {

    public static  MetricRegistry METRIC_REGISTRY = new MetricRegistry();

    static {
    	Counter c = new Counter();
    	c.inc();
    	METRIC_REGISTRY.register("my_counter{this=that}", c);
    	METRIC_REGISTRY.register("my_meter{this=that}", new Meter());

    	METRIC_REGISTRY.register("my_timer", new Timer());
    	METRIC_REGISTRY.histogram("my_histogram");
    	METRIC_REGISTRY.register("my_gauge", new Gauge<Integer>() {

			@Override
			public Integer getValue() {
				// TODO Auto-generated method stub
				return null;
			}

		});

    }

    @Override
    protected MetricRegistry getMetricRegistry() {
        return METRIC_REGISTRY;
    }


}
