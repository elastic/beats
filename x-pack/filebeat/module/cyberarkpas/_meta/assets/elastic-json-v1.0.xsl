<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:import href="./Syslog/RFC5424Changes.xsl"/>
<xsl:output method='text' version='1.0' encoding='UTF-8' indent='no'/>

<!-- version control variables -->
<xsl:variable name="format" select="'elastic'"/>
<xsl:variable name="version" select="'1.0'"/>

<!-- configuration -->
<xsl:variable name="include_raw" select="0"/> <!-- save a "raw" key with the original XML -->

<!-- main object with header info -->
<xsl:template match="/">
  <xsl:apply-imports/>
  <xsl:text>{"format":"</xsl:text><xsl:value-of select="$format"/>
  <xsl:text>","version":"</xsl:text><xsl:value-of select="$version"/>
  <xsl:text>"</xsl:text>
  <xsl:choose>
    <xsl:when test="$include_raw=1">
      <xsl:text>,"raw":</xsl:text>
      <xsl:call-template name="json-string">
        <xsl:with-param name="text">
          <xsl:apply-templates select="*" mode="raw"/>
        </xsl:with-param>
      </xsl:call-template>
    </xsl:when>
  </xsl:choose>
  <xsl:text>,</xsl:text>
  <xsl:apply-templates select="*" mode="object"/>
  <!-- this text below includes the terminating newline -->
  <xsl:text>}&#xa;</xsl:text>
</xsl:template>

<xsl:template match="*" mode="raw">
    <xsl:value-of select="concat('&lt;', name())" />
    <xsl:for-each select="@*">
      <xsl:value-of select="concat(' ', name(), '=&quot;', ., '&quot;')"/>
    </xsl:for-each>
    <xsl:text>&gt;</xsl:text>
    <xsl:apply-templates mode="raw"/>
    <xsl:value-of select="concat('&lt;/', name(), '&gt;')" />
  </xsl:template>

<!-- serialize objects -->
<xsl:template match="*" mode="object">
  <xsl:text>&quot;</xsl:text>
  <xsl:value-of select="name()"/><xsl:text>&quot;:</xsl:text><xsl:call-template name="value">
        <xsl:with-param name="parent" select="1"/>
    </xsl:call-template>
</xsl:template>

<!-- serialize array elements -->
<xsl:template match="*" mode="array">
    <xsl:call-template name="value"/>
</xsl:template>

<!-- value of node serializer -->
<xsl:template name="value">
  <xsl:param name="parent"/>
  <xsl:variable name="childName" select="name(*[1])"/>
  <xsl:choose>
    <xsl:when test="not(*|@*)">
      <xsl:choose>
        <xsl:when test="$parent=1">
          <xsl:call-template name="json-string">
            <xsl:with-param name="text" select="."/>
          </xsl:call-template>
          </xsl:when>
        <xsl:otherwise>
          <xsl:call-template name="json-string">
            <xsl:with-param name="text" select="name()"/>
          </xsl:call-template>
          <xsl:text>:</xsl:text>
          <xsl:call-template name="json-string">
            <xsl:with-param name="text" select="."/>
          </xsl:call-template>
          </xsl:otherwise>
      </xsl:choose>
    </xsl:when>
    <xsl:when test="count(*[name()=$childName]) > 1">
      <xsl:text>{</xsl:text>
      <xsl:call-template name="json-string">
        <xsl:with-param name="text" select="$childName"/>
      </xsl:call-template>
      <xsl:text>:[</xsl:text>
      <xsl:apply-templates select="*" mode="array"/>
      <xsl:text>]}</xsl:text>
      </xsl:when>
    <xsl:otherwise>
      <xsl:text>{</xsl:text>
        <xsl:apply-templates select="@*" mode="attrs"/>
        <xsl:if test='count(@*)>0 and count(*)>0'>,</xsl:if>
        <xsl:apply-templates select="*" mode="object"/>
      <xsl:text>}</xsl:text>
    </xsl:otherwise>
  </xsl:choose>
  <xsl:if test="following-sibling::*"><xsl:text>,</xsl:text></xsl:if>
</xsl:template>

<!-- serialize attributes -->
<xsl:template match="@*" mode="attrs">
    <xsl:call-template name="json-string">
      <xsl:with-param name="text" select="name()"/>
    </xsl:call-template>
    <xsl:text>:</xsl:text>
    <xsl:call-template name="json-string">
      <xsl:with-param name="text" select="."/>
    </xsl:call-template>
    <xsl:if test="position()!=last()"><xsl:text>,</xsl:text></xsl:if>
</xsl:template>

<!-- json-string converts a text to a quoted and escaped JSON string -->
<xsl:template name="json-string">
  <xsl:param name="text"/>
  <xsl:variable name="tmp">
      <xsl:call-template name="string-replace">
        <xsl:with-param name="string" select="$text"/>
        <xsl:with-param name="from" select="'\'"/>
        <xsl:with-param name="to" select="'\\'"/>
      </xsl:call-template>
  </xsl:variable>
  <xsl:variable name="tmp2">
    <xsl:call-template name="string-replace">
      <xsl:with-param name="string" select="$tmp"/>
      <xsl:with-param name="from" select="'&#xa;'"/>
      <xsl:with-param name="to" select="'\n'"/>
    </xsl:call-template>
  </xsl:variable>
  <xsl:variable name="tmp3">
    <xsl:call-template name="string-replace">
      <xsl:with-param name="string" select="$tmp2"/>
      <xsl:with-param name="from" select="'&#xd;'"/>
      <xsl:with-param name="to" select="'\r'"/>
    </xsl:call-template>
  </xsl:variable>
  <xsl:variable name="tmp4">
    <xsl:call-template name="string-replace">
      <xsl:with-param name="string" select="$tmp3"/>
      <xsl:with-param name="from" select="'&#x09;'"/>
      <xsl:with-param name="to" select="'\t'"/>
    </xsl:call-template>
  </xsl:variable>
  <xsl:text>&quot;</xsl:text>
  <xsl:call-template name="string-replace">
    <xsl:with-param name="string" select="$tmp4"/>
    <xsl:with-param name="from" select="'&quot;'"/>
    <xsl:with-param name="to" select="'\&quot;'"/>
  </xsl:call-template>
  <xsl:text>&quot;</xsl:text>
</xsl:template>

<!-- replace all occurences of the character(s) `from'
 by the string `to' in the string `string'.-->
<xsl:template name="string-replace">
  <xsl:param name="string"/>
  <xsl:param name="from"/>
  <xsl:param name="to"/>
  <xsl:choose>
    <xsl:when test="contains($string,$from)">
      <xsl:value-of select="substring-before($string,$from)"/>
      <xsl:value-of select="$to"/>
      <xsl:call-template name="string-replace">
        <xsl:with-param name="string" select="substring-after($string,$from)"/>
        <xsl:with-param name="from" select="$from"/>
        <xsl:with-param name="to" select="$to"/>
      </xsl:call-template>
    </xsl:when>
    <xsl:otherwise>
      <xsl:value-of select="$string"/>
    </xsl:otherwise>
  </xsl:choose>
</xsl:template>
 
</xsl:stylesheet>
